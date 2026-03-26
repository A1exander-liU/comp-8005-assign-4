// Package worker contains class for worker
package worker

import (
	"context"
	"encoding/gob"
	"errors"
	"flag"
	"fmt"
	"io"
	"net"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/A1exander-liU/comp-8005-assign-2/internal/shared"
	"github.com/go-crypt/crypt"
	"go.uber.org/zap"
)

// Config holds parameters for worker setup:
type Config struct {
	// IP address of the controller to connect to
	ControllerIP string

	// Port number of the controller to connect to
	ControllerPort int

	// Number of threads to use for password cracking
	Threads int
}

// Threads to communicate individual cracking results
type doneResp struct {
	found    bool
	password string
	err      error
}

// Worker is reponsible for receiving password cracking jobs from
// the controller and sending the results back.
type Worker struct {
	// Public logger for sending log messages
	Logger *zap.Logger

	// Worker id, IP:PORT format
	ID string

	Threads int

	router  *shared.Router
	conn    net.Conn
	encoder *gob.Encoder
	decoder *gob.Decoder

	state            WorkerState
	totalAttempts    int
	lastAttemptsSent int
	mu               sync.Mutex

	Config Config

	fs *flag.FlagSet
}

// NewWorker creates a new worker with the provided logger instance.
func NewWorker(logger *zap.Logger) *Worker {
	return &Worker{
		Logger:        logger,
		totalAttempts: 0,
	}
}

// SetupServer creates a connection with the controller.
//
// This will exit with an error if it failed to connect.
func (w *Worker) SetupServer() {
	address := net.JoinHostPort(w.Config.ControllerIP, strconv.Itoa(w.Config.ControllerPort))

	conn, err := net.Dial("tcp", address)
	if err != nil {
		fmt.Printf("Failed to connected to %s: %s\n", address, err)
		os.Exit(1)
	}

	w.Logger.Info(fmt.Sprintf("Connected to controller at %s", address))
	w.ID = conn.LocalAddr().String()
	w.Threads = w.Config.Threads
	w.conn = conn
	w.encoder = gob.NewEncoder(w.conn)
	w.decoder = gob.NewDecoder(w.conn)
}

// Send a message to the controller
func (w *Worker) send(m shared.Message) error {
	err := w.encoder.Encode(m)
	if err != nil {
		w.Logger.Error("Failed to send", zap.Error(err))
		return err
	}

	w.Logger.Info(m.Message)
	return nil
}

func (w *Worker) sendRegistration() error {
	m := shared.Message{
		Version:   shared.MessageVersion,
		ID:        w.ID,
		Type:      shared.MessageRegister,
		Timestamp: time.Now(),
		Message:   "Sending registration request",
	}

	return w.send(m)
}

func (w *Worker) sendJobRequest() error {
	m := shared.Message{
		Version:   shared.MessageVersion,
		ID:        w.ID,
		Type:      shared.MessageJobDetails,
		Timestamp: time.Now(),
		Message:   "Sending request for job",
	}

	return w.send(m)
}

func (w *Worker) sendJobResults(result string, err error, crackTime, dispatchTime time.Duration, chunkID int) error {
	m := shared.Message{
		Version:   shared.MessageVersion,
		ID:        w.ID,
		Type:      shared.MessageJobResults,
		Timestamp: time.Now(),
	}

	jobResults := shared.PayloadJobResults{}
	if err != nil {
		w.Logger.Info("Failed to crack password")

		m.Message = fmt.Sprintf("Sending job results: %s", err.Error())
		jobResults.Password = ""
		jobResults.CrackTime = crackTime
		jobResults.DispatchTime = dispatchTime
		jobResults.Err = err.Error()
		jobResults.ChunkID = chunkID
	} else {
		w.Logger.Info("Sucessfully cracked password")

		m.Message = "Sending job results: Password cracked"
		jobResults.Password = result
		jobResults.CrackTime = crackTime
		jobResults.DispatchTime = dispatchTime
		jobResults.Err = ""
		jobResults.ChunkID = chunkID
	}
	m.Payload = jobResults

	return w.send(m)
}

func (w *Worker) sendClose() error {
	m := shared.Message{
		Version:   shared.MessageVersion,
		ID:        w.ID,
		Type:      shared.MessageClose,
		Message:   "Sending close request",
		Timestamp: time.Now(),
	}

	return w.send(m)
}

// Sending back heartbeat messages
func (w *Worker) handleHeartbeat() error {
	total := w.getTotalAttempts()
	delta := total - w.lastAttemptsSent

	payload := shared.PayloadHearbeat{TotalTested: total, DeltaTested: delta, ActiveThreads: w.Threads}
	m := shared.Message{
		Version:   shared.MessageVersion,
		ID:        w.ID,
		Type:      shared.MessageHeartbeat,
		Message:   "Sending heartbeat",
		Timestamp: time.Now(),
		Payload:   payload,
	}

	w.lastAttemptsSent = total

	return w.send(m)
}

func (w *Worker) incTotalAttempts() {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.totalAttempts += 1
}

func (w *Worker) getTotalAttempts() int {
	var attempts int

	w.mu.Lock()
	defer w.mu.Unlock()

	attempts = w.totalAttempts
	return attempts
}

func (w *Worker) handleJob(payload shared.PayloadJobDetails, dispatchTime time.Duration) {
	passwords := []string{}
	for i := payload.ChunkStart; i < payload.ChunkEnd; i++ {
		passwords = append(passwords, shared.EncodeBase(i, shared.SearchSpace))
	}

	partitions := shared.PartitionArray(passwords, w.Threads)
	fullHash := w.buildHash(payload)

	done := make(chan doneResp, 1)
	cancels := []chan bool{}

	var dr doneResp

	var wg sync.WaitGroup

	// Container
	wg.Go(func() {
		threadsDone := 0

		for {
			select {
			case d := <-done:
				dr = d
				threadsDone += 1
				// Cancel existing threads once password is found
				if d.found {
					for _, cancel := range cancels {
						cancel <- true
						close(cancel)
					}
					return
				}

			default:
				if threadsDone == w.Threads {
					return
				}
			}
		}
	})

	// Threads
	start := time.Now()

	for id, partition := range partitions {
		cancel := make(chan bool, 1)
		cancels = append(cancels, cancel)

		wg.Go(func() {
			w.Logger.Info(fmt.Sprintf("Thread %d started", id+1))
			decoder, _ := crypt.NewDecoderAll()
			for _, password := range partition {
				w.incTotalAttempts()

				select {
				case <-cancel:
					return
				default:
				}

				digest, err := decoder.Decode(fullHash)
				if err != nil {
					done <- doneResp{found: false, password: "", err: err}
					return
				}
				match := digest.Match(password)

				if match {
					done <- doneResp{found: true, password: password, err: nil}
					return
				}
			}
			// password not found in this chunk
			done <- doneResp{found: false, password: "", err: errors.New("failed to find password in chunk")}
		})
	}

	wg.Wait()
	err := w.sendJobResults(dr.password, dr.err, time.Since(start), dispatchTime, payload.ChunkID)
	if err != nil {
		w.Logger.Error("Failed to send job results", zap.Error(err))
		w.cleanup()
	}
}

func (w *Worker) cleanup() {
	_ = w.Logger.Sync()
	_ = w.conn.Close()
}

func (w *Worker) Start() {
	ctx, cancel := context.WithCancel(context.Background())
	router := shared.NewRouter(w.Logger, w.conn)

	router.Handle(shared.MessageRegister, w.routeRegister)
	router.Handle(shared.MessageReconnect, w.routeReconnect)
	router.Handle(shared.MessageJobDetails, w.routeJobDetails)
	router.Handle(shared.MessageJobResults, w.routeJobResults)
	router.Handle(shared.MessageHeartbeat, w.routeHeartbeat)
	router.Handle(shared.MessageClose, w.routeClose)

	w.router = router

	// load existing state if any and reconnect
	// if loading state or reconnection fails, registration is done instead

	state, err := LoadState("./data/data.json")
	w.state = state

	if err != nil {
		w.Logger.Warn("Failed to load state", zap.Error(err))
		w.router.Send(shared.Message{
			Version:   shared.MessageVersion,
			Type:      shared.MessageRegister,
			Timestamp: time.Now(),
			Message:   "Registration request",
		})
	} else {
		w.router.Send(shared.Message{
			Version:   shared.MessageVersion,
			Type:      shared.MessageReconnect,
			Timestamp: time.Now(),
			Message:   "Reconnection request",
			Payload:   shared.PayloadReconnect{ID: state.ID},
		})
	}

	if err := router.Start(ctx, cancel); err != nil {
		w.cleanup()
	}
}

// HandleConnection handles worker lifecycle of sending and receiving to and from the controller.
func (w *Worker) HandleConnection() {
	// cracking job goes into thread
	// loop still keeps processing messages (i.e. heartbeat while cracking)
	// how can the cracking threads notify when done without blocking
	// - use a channel to notify

	err := w.sendRegistration()
	if err != nil {
		w.cleanup()
		return
	}

outer:
	for {
		var m shared.Message

		err := w.decoder.Decode(&m)
		if err == io.EOF {
			w.Logger.Info("connection closed")
			break outer
		}
		if err != nil {
			w.Logger.Error("Failed to decode", zap.Error(err))
			break outer
		}

		w.Logger.Info("Received from controller", zap.String("message", m.Message))

		switch m.Type {
		case shared.MessageRegister:
			err := w.sendJobRequest()
			if err != nil {
				break outer
			}

		case shared.MessageJobDetails:
			payload, _ := m.Payload.(shared.PayloadJobDetails)
			w.Logger.Info("Received job details", zap.String("algorithm", payload.Algorithm), zap.Int("chunk", payload.ChunkID))

			// no more available chunks: exit
			if payload.ChunkID == -1 {
				w.Logger.Info("No more available tasks: exiting")
				if err := w.sendClose(); err != nil {
					break outer
				}
				break
			}

			// Start cracking here
			w.Logger.Info("Cracking password...")
			go w.handleJob(payload, time.Since(m.Timestamp))

		case shared.MessageJobResults:
			payload, _ := m.Payload.(shared.PayloadJobResultsResp)

			var err error
			if payload.Done {
				err = w.sendClose()
			} else {
				err = w.sendJobRequest()
			}
			if err != nil {
				break outer
			}

		case shared.MessageHeartbeat:
			err := w.handleHeartbeat()
			if err != nil {
				break outer
			}

		case shared.MessageClose:
			break outer
		}
	}

	w.Logger.Info("initiating cleanup")
	w.cleanup()
}
