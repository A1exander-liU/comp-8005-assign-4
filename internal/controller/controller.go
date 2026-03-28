// Package controller contains class for controller
//
// This consists of:
//   - cli which has arg parsing related logic
//   - display for printing timing results
//   - jobs for handling assigning jobs to workers
//   - routes which are handlers for different message types
package controller

import (
	"context"
	"flag"
	"fmt"
	"net"
	"os"
	"sync"
	"time"

	"github.com/A1exander-liU/comp-8005-assign-2/internal/shared"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

type ChunkStatus string

const (
	ChunkUnassigned = "unassigned"
	ChunkAssigned   = "assigned"
	ChunkCompleted  = "completed"
)

// Config holds parameters for controller setup.
type Config struct {
	// Path to the shadowfile
	Shadowfile string

	// User's name to crack
	Username string

	// Port number for the controller to listen on
	Port int

	// Period for sending a heartbeat
	HeartbeatSeconds int

	// Cracking job size for workers
	ChunkSize int

	// Number of passwords attempts before worker should send a checkpoint
	CheckpointAttempts int
}

type workerConnection struct {
	// Only registered can be sent jobs
	Registered bool

	Connected     bool
	reconnectionC chan bool

	// Current chunk its working on, -1 indicates not working
	ChunkID int

	Conn   net.Conn
	Router *shared.Router

	ctx    context.Context
	cancel context.CancelFunc

	HeartbeatsSinceReply int

	// Shared channel for router hook
	incomingMessages chan shared.Message
}

type chunk struct {
	// Start and end index of the passwords (end is exclusive)
	start, end, index uint64

	// Assigned means active worker, so can't be assigned
	// Unassigned means no active worker, so can be assigned
	// Compeleted means all passwords in chunk tried, so can't be assigned
	status ChunkStatus
}

type chunkTiming struct {
	// durations
	dispatchTime, chunkAssignTime, crackTime, returnTime time.Duration

	// start times for recording across function calls if required
	dispatchStart, chunkAssignStart, returnTimeStart time.Time
}

// Controller is reponsible for managing worker connections
// for sending and receiving password cracking jobs.
type Controller struct {
	// Public logger for sending log messages
	Logger *zap.Logger

	listener net.Listener
	workers  map[string]*workerConnection
	fs       *flag.FlagSet

	// starting id of next chunk to generate
	// id 0 = 0 * chunk size
	// id 10 = 10 * chunk size
	nextChunkIDMutex sync.Mutex
	nextChunkID      int

	// ChunkID to chunk data
	chunks map[int]*chunk

	chunkTimings map[int]*chunkTiming
	deltaTimings []int
	crackStart   *time.Time

	ShadowData shared.ShadowData
	Config     Config

	LatencyParse        time.Duration
	LatencyDispatch     time.Duration
	LatencyDispatchTime time.Time
	LatencyCrack        time.Duration
	LatencyReturn       time.Duration
}

// NewController creates a new Controller object.
func NewController(logger *zap.Logger) *Controller {
	return &Controller{
		Logger: logger,

		workers:      map[string]*workerConnection{},
		chunks:       map[int]*chunk{},
		chunkTimings: map[int]*chunkTiming{},
		deltaTimings: make([]int, 0),
		crackStart:   nil,
	}
}

// handleWorkerCrashes is called everytime a worker exits. Checks
func (c *Controller) handleWorkerCrashes() {
	for id, worker := range c.workers {
		if worker.Connected {
			continue
		}

		if worker.ChunkID == -1 {
			continue
		}

		// only attempt to revoke jobs from workers that exited with an assigned job
		c.Logger.Info("Worker disconnected with work", zap.String("workerID", id), zap.Int("chunkID", worker.ChunkID))
		go func() {
			timer := time.NewTimer(time.Duration(c.Config.HeartbeatSeconds) * time.Second)

			for {
				select {
				case <-timer.C:
					c.Logger.Info("Worker did not reconnect in time", zap.String("workerID", id))
					c.revokeJob(id, worker.ChunkID)
					return
				case <-worker.reconnectionC:
					timer.Stop()
					c.Logger.Info("Worker reconnected in time", zap.String("workerID", id))
					return
				}
			}
		}()
	}
}

// getUnassignedChunk gives a chunk of passwords that hasn't been assigned to another worker.
//
// An existing unassigned chunk will be searched for first before creating a new chunk.
//
// It returns the chunk id and a boolean if the chunk was assigned.
//
// In the case, a chunk couldn't be assigned, this will return a chunk id of -1 and false.
func (c *Controller) getUnassignedChunk(workerID string) (int, bool) {
	// assign existing chunk first
	c.nextChunkIDMutex.Lock()
	for chunkID, chunk := range c.chunks {
		if chunk.status != ChunkUnassigned {
			continue
		}

		c.assignJob(workerID, chunkID)
		c.nextChunkIDMutex.Unlock()
		return chunkID, true
	}
	c.nextChunkIDMutex.Unlock()

	var nextID int
	c.nextChunkIDMutex.Lock()
	nextID = c.nextChunkID
	c.nextChunkID += 1
	c.nextChunkIDMutex.Unlock()

	c.assignJob(workerID, nextID)

	return nextID, true
}

// SetupServer starts listening for connections on the specified port.
//
// This will exit with an error if listening failed.
func (c *Controller) SetupServer() {
	address := fmt.Sprintf("[::]:%d", c.Config.Port)

	listener, err := net.Listen("tcp", address)
	if err != nil {
		fmt.Printf("Failed to listen on %s: %s\n", address, err)
		os.Exit(1)
	}

	c.listener = listener

	c.Logger.Info(fmt.Sprintf("Controller listening on %s", address))
}

// AcceptConnection accepts an incoming connection.
//
// This wraps around the net.Listener.Accept
func (c *Controller) AcceptConnection() (net.Conn, error) {
	return c.listener.Accept()
}

// processHeartbeat handles sending heartbeats to workers to determine their liveliness.
//
// A heartbeat is sent periodically based on the heartbeat seconds if the worker is actively
// working on a job. It keeps a counter of heartbeats sent, resetting when it receives a reply.
//
// If the worker doesn't respond before a subsequent heartbeat (heartbeats since reply > 1), the
// worker is identified as unresponsive, and their current job will be revoked. Any job results
// or checkpoints sent will be rejected; the worker must request a new job.
func (c *Controller) processHeartbeat(workerID string) {
	ticker := time.NewTicker(time.Duration(c.Config.HeartbeatSeconds) * time.Second)
	worker := c.workers[workerID]

	for {
		select {
		case <-worker.ctx.Done():
			worker.Connected = false
			go c.handleWorkerCrashes()
			ticker.Stop()
			return

		case <-ticker.C:
			if worker.ChunkID == -1 {
				continue
			}

			m := shared.Message{
				Version:   shared.MessageVersion,
				ID:        workerID,
				Type:      shared.MessageHeartbeat,
				Timestamp: time.Now(),
				Message:   "Heartbeat from controller",
			}
			worker.Router.Send(m)
			worker.HeartbeatsSinceReply += 1

		case m := <-worker.incomingMessages:
			// only care about heartbeats
			if m.Type != shared.MessageHeartbeat {
				continue
			}

			payload := m.Payload.(shared.PayloadHearbeat)

			c.Logger.Info("Receieved heartbeat", zap.Int("total", payload.TotalTested), zap.Int("delta", payload.DeltaTested))
			c.deltaTimings = append(c.deltaTimings, payload.DeltaTested)

			// failed to respond to heartbeat in time, revoke the job
			if worker.HeartbeatsSinceReply > 1 {
				message := fmt.Sprintf("worker %s failed to respond for too many heartbeart", workerID)

				c.Logger.Warn(message)
				c.revokeJob(workerID, worker.ChunkID)
			}

			worker.HeartbeatsSinceReply = 0
		}
	}
}

// sendStop sends a stop signal to every registered worker to request
// every worker to terminate.
//
// This will additionally unregister all existing worker connections
// from the controller.
func (c *Controller) sendStop() {
	for id, worker := range c.workers {
		worker.Router.Send(shared.Message{
			Version:   shared.MessageVersion,
			ID:        id,
			Type:      shared.MessageClose,
			Timestamp: time.Now(),
			Message:   "Sending close",
		})
	}

	c.Logger.Info("Closing controller")
	if err := c.listener.Close(); err != nil {
		c.Logger.Error("Failed to close controller", zap.Error(err))
	}
}

// HandleConnection manages communication with a single worker for the
// whole entire lifecycle.
func (c *Controller) HandleConnection(conn net.Conn) {
	id := uuid.NewString()

	r := shared.NewRouterWithID(c.Logger, conn, id)
	r.Handle(shared.MessageRegister, c.handleRegistration)
	r.Handle(shared.MessageJobDetails, c.sendJob)
	r.Handle(shared.MessageJobCheckpoint, c.handleJobCheckpoint)
	r.Handle(shared.MessageJobResults, c.handleJobResults)
	r.Handle(shared.MessageClose, c.handleClose)
	r.Handle(shared.MessageReconnect, c.handleReconnect)

	incomingMessages := make(chan shared.Message, 10)
	r.HookRead(incomingMessages)

	ctx, cancel := context.WithCancel(context.Background())
	c.workers[id] = &workerConnection{
		Registered:    false,
		Connected:     true,
		reconnectionC: make(chan bool, 1),
		Conn:          conn, Router: r,
		ctx: ctx, cancel: cancel,
		incomingMessages: incomingMessages,
	}

	go c.processHeartbeat(id)

	if err := r.Start(ctx, cancel); err != nil {
		c.Logger.Error(err.Error())
	}

	// worker connection ended
	// cleanup code goes down here
}
