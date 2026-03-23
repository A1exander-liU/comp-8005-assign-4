// Package controller contains class for controller
package controller

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"net"
	"os"
	"sync"
	"time"

	"github.com/A1exander-liU/comp-8005-assign-2/internal/shared"
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
	start, end uint64

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

// getUnassignedChunk gives a chunk of passwords that hasn't been assigned to another worker.
//
// It will return the chunk id and a boolean if the chunk was assigned.
// In the case that there are no more unassigned chunks, this will return
// a chunk id of -1 and false
func (c *Controller) getUnassignedChunk(workerID string) (int, bool) {
	// assign existing chunk first
	c.nextChunkIDMutex.Lock()
	for chunkID, chunk := range c.chunks {
		if chunk.status != ChunkUnassigned {
			continue
		}

		c.assignJob(workerID, chunkID)
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

// handleRegistration updates worker connection information by setting the
// registered flag.
//
// An `MessageError` will be returned if worker already registered.
func (c *Controller) handleRegistration(m shared.Message, conn net.Conn) (shared.Message, error) {
	id := conn.RemoteAddr().String()
	ok := c.workers[id].Registered
	if ok {
		err := fmt.Errorf("worker %s is already registered", id)
		return shared.Message{ID: id, Type: shared.MessageError, Timestamp: time.Now(), Message: err.Error()}, err
	}

	c.workers[id].Registered = true

	c.LatencyDispatchTime = time.Now()

	return shared.Message{
			Version:   shared.MessageVersion,
			ID:        id,
			Type:      shared.MessageRegister,
			Timestamp: time.Now(),
			Message:   "Registration successful",
			Payload: shared.PayloadRegisterResp{
				HeartbeatSeconds:   c.Config.HeartbeatSeconds,
				CheckpointAttempts: c.Config.CheckpointAttempts,
			},
		},
		nil
}

// sendJob handles sending job details to a worker, only registered workers can receive job details.
func (c *Controller) sendJob(_ shared.Message, conn net.Conn) (shared.Message, error) {
	id := conn.RemoteAddr().String()
	if _, ok := c.workers[id]; !ok {
		err := fmt.Errorf("worker %s is not registered", id)
		return shared.Message{
				Version:   shared.MessageVersion,
				ID:        id,
				Type:      shared.MessageError,
				Timestamp: time.Now(),
				Message:   err.Error(),
			},
			err
	}

	timestamp := time.Now()
	if c.crackStart == nil {
		c.crackStart = &timestamp
	}

	chunkID, _ := c.getUnassignedChunk(id)
	c.chunkTimings[chunkID] = &chunkTiming{
		dispatchStart:    timestamp,
		chunkAssignTime:  time.Since(timestamp),
		chunkAssignStart: timestamp,
	}

	res := shared.Message{
		Version: shared.MessageVersion, Type: shared.MessageJobDetails, Message: "Cracking details",
		Timestamp: timestamp,
		Payload: shared.PayloadJobDetails{
			Algorithm:  c.ShadowData.Algorithm,
			Parameters: c.ShadowData.Parameters,
			Salt:       c.ShadowData.Salt,
			Hash:       c.ShadowData.Hash,
			ChunkID:    chunkID,
			ChunkStart: c.chunks[chunkID].start,
			ChunkEnd:   c.chunks[chunkID].end,
		},
	}

	c.LatencyDispatch = timestamp.Sub(c.LatencyDispatchTime)

	return res, nil
}

func (c *Controller) handleJobCheckpoint(m shared.Message, conn net.Conn) (shared.Message, error) {
	id := conn.RemoteAddr().String()

	payload, ok := m.Payload.(shared.PayloadCheckpoint)
	if !ok {
		err := fmt.Errorf("expected PayloadCheckpoint")
		return shared.Message{
				Version:   shared.MessageVersion,
				ID:        id,
				Type:      shared.MessageError,
				Timestamp: time.Now(),
				Message:   err.Error(),
			},
			err
	}

	c.Logger.Info("received checkpoint", zap.String("worker", id), zap.Int("chunkID", payload.ChunkID))

	return shared.Message{}, nil
}

func (c *Controller) handleJobResults(m shared.Message, conn net.Conn) (shared.Message, error) {
	id := conn.RemoteAddr().String()
	worker, ok := c.workers[id]

	// check if worker is registered
	if !ok {
		err := fmt.Errorf("worker %s not registered", id)
		return shared.Message{
				Version:   shared.MessageVersion,
				Type:      shared.MessageError,
				Timestamp: time.Now(),
				Message:   err.Error(),
			},
			nil
	}

	payload, ok := m.Payload.(shared.PayloadJobResults)
	if !ok {
		return shared.Message{Version: shared.MessageVersion, Type: shared.MessageError, Message: "Bad payload"}, nil
	}

	// check if worker is assigned to the job
	if worker.ChunkID != payload.ChunkID {
		err := fmt.Errorf("worker %s is not assigned to chunk %d", id, payload.ChunkID)
		return shared.Message{
				Version:   shared.MessageVersion,
				Type:      shared.MessageError,
				Timestamp: time.Now(),
				Message:   err.Error(),
			},
			nil
	}

	timestamp := time.Now()

	c.chunkTimings[payload.ChunkID].crackTime = payload.CrackTime
	c.chunkTimings[payload.ChunkID].dispatchTime = payload.DispatchTime.Abs()
	c.chunkTimings[payload.ChunkID].returnTime = time.Since(m.Timestamp).Abs()

	var err error
	var done bool
	if payload.Err != "" {
		err = errors.New(payload.Err)
		done = false
	} else {
		err = nil
		done = true
	}

	c.chunks[payload.ChunkID].status = ChunkCompleted
	c.workers[conn.RemoteAddr().String()].ChunkID = -1

	res := shared.Message{
		Version:   shared.MessageVersion,
		Type:      shared.MessageJobResults,
		Message:   "Received message details",
		Timestamp: time.Now(),
		Payload:   shared.PayloadJobResultsResp{Done: done},
	}

	c.LatencyCrack = payload.CrackTime
	c.LatencyReturn = m.Timestamp.Sub(timestamp)

	c.displayJobResults(payload.Password, err, payload.ChunkID, timestamp)

	if err == nil {
		c.sendStop()
	}

	return res, nil
}

// handleClose performs cleanup logic after a worker requests to close their connection.
//
// This will delete their entry as a worker and also reclaim any ongoing work they have.
func (c *Controller) handleClose(_ shared.Message, conn net.Conn) (shared.Message, error) {
	id := conn.RemoteAddr().String()
	_, ok := c.workers[id]
	if ok {
		chunkToReclaim := c.workers[id].ChunkID
		if chunkToReclaim != -1 {
			c.chunks[chunkToReclaim].status = ChunkUnassigned
		}

		delete(c.workers, id)
	}

	message := fmt.Sprintf("Closing connection for %s", id)
	return shared.Message{Version: shared.MessageVersion, ID: id, Type: shared.MessageClose, Timestamp: time.Now(), Message: message}, nil
}

func (c *Controller) displayJobResults(result string, err error, chunkID int, ts time.Time) {
	startPassword := shared.EncodeBase(c.chunks[chunkID].start, shared.SearchSpace)
	endPassword := shared.EncodeBase(c.chunks[chunkID].end, shared.SearchSpace)
	timings := c.chunkTimings[chunkID]

	var passwordString string
	chunkString := fmt.Sprintf("==== CHUNK: '%s' to '%s' RESULTS (seconds) ====", startPassword, endPassword)

	if err != nil {
		passwordString = fmt.Sprintf("PASSWORD NOT FOUND: %s", err)
	} else {
		passwordString = fmt.Sprintf("PASSWORD: %s", result)
	}

	c.prettyPrintResults(
		chunkString,
		passwordString,
		c.LatencyParse,
		timings.dispatchTime,
		timings.chunkAssignTime,
		timings.crackTime,
		timings.returnTime,
	)

	// report final results if password found
	if err != nil {
		return
	}

	finalString := "==== FINAL RESULTS (seconds) ===="
	var totaldispatch, totalChunkAssign, totalCrack, totalReturn time.Duration
	for _, timing := range c.chunkTimings {
		totaldispatch += timing.dispatchTime
		totalChunkAssign += timing.chunkAssignTime
		totalCrack += timing.crackTime
		totalReturn += timing.returnTime
	}
	c.prettyPrintResults(
		finalString,
		passwordString,
		c.LatencyParse,
		totaldispatch,
		totalChunkAssign,
		ts.Sub(*c.crackStart),
		totalReturn,
	)

	delta := 0
	for _, d := range c.deltaTimings {
		delta += d
	}
	averageDelta := float64(delta) / float64(len(c.deltaTimings))
	fmt.Printf("Average Delta (heartbeat/%ds): %f\n", c.Config.HeartbeatSeconds, averageDelta)
}

func (c *Controller) prettyPrintResults(
	title, password string,
	parse, dispatch, chunkAssign, crack, returnTime time.Duration,
) {
	total := parse + dispatch + chunkAssign + crack + returnTime

	fmt.Println(title)
	fmt.Println(password)
	fmt.Println("Parse:", parse.Seconds())
	fmt.Println("Dispatch:", dispatch.Seconds())
	fmt.Println("ChunkAssign:", chunkAssign.Seconds())
	fmt.Println("Crack:", crack.Seconds())
	fmt.Println("Return:", returnTime.Seconds())
	fmt.Println("Total:", total.Seconds())
	fmt.Println("=================================")
	fmt.Println()
}

func (c *Controller) processHeartbeat(workerID string) {
	ticker := time.NewTicker(time.Duration(c.Config.HeartbeatSeconds) * time.Second)
	worker := c.workers[workerID]

	for {
		if worker.ChunkID == -1 {
			continue
		}

		select {
		case <-worker.ctx.Done():
			ticker.Stop()
			return

		case <-ticker.C:
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
				message := fmt.Sprintf("worker %s failed to respond to heartbeart", workerID)

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
	for _, worker := range c.workers {
		id := worker.Conn.RemoteAddr().String()

		_, ok := c.workers[id]
		if ok {
			delete(c.workers, id)
		}

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
	r := shared.NewRouter(c.Logger, conn)
	r.Handle(shared.MessageRegister, c.handleRegistration)
	r.Handle(shared.MessageJobDetails, c.sendJob)
	r.Handle(shared.MessageJobCheckpoint, c.handleJobCheckpoint)
	r.Handle(shared.MessageJobResults, c.handleJobResults)
	r.Handle(shared.MessageClose, c.handleClose)

	incomingMessages := make(chan shared.Message, 10)
	r.HookRead(incomingMessages)

	go c.processHeartbeat(conn.RemoteAddr().String())

	ctx, cancel := context.WithCancel(context.Background())
	c.workers[conn.RemoteAddr().String()] = &workerConnection{
		Registered: false,
		Conn:       conn, Router: r,
		ctx: ctx, cancel: cancel,
		incomingMessages: incomingMessages,
	}

	if err := r.Start(ctx, cancel); err != nil {
		c.Logger.Error(err.Error())
	}

	// worker connection ended
	// cleanup code goes down here
}
