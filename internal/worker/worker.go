// Package worker contains class for worker
package worker

import (
	"context"
	"encoding/gob"
	"flag"
	"fmt"
	"net"
	"os"
	"os/signal"
	"strconv"
	"sync"
	"syscall"
	"time"

	"github.com/A1exander-liU/comp-8005-assign-2/internal/shared"
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

func (w *Worker) cleanup() {
	_ = w.Logger.Sync()
	_ = w.conn.Close()
}

func (w *Worker) Start() {
	ctx, cancel := context.WithCancel(context.Background())

	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		for range shutdown {
			_ = SaveState(StateFileLocation, w.state)
			cancel()
			return
		}
	}()

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

	state, err := LoadState(StateFileLocation)

	if err != nil {
		w.Logger.Warn("Failed to load state", zap.Error(err))
		w.router.Send(shared.Message{
			Version:   shared.MessageVersion,
			Type:      shared.MessageRegister,
			Timestamp: time.Now(),
			Message:   "Registration request",
		})
	} else {
		w.state = state
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
