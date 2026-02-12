// Package controller contains class for controller
package controller

import (
	"fmt"
	"net"
	"os"
	"time"

	"github.com/A1exander-liU/comp-8005-assign-2/internal/shared"
	"go.uber.org/zap"
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
}

type workerConnection struct {
	Registered bool
	Done       chan bool

	Conn   net.Conn
	Router *shared.Router
}

// Controller is reponsible for managing worker connections
// for sending and receiving password cracking jobs.
type Controller struct {
	// Public logger for sending log messages
	Logger *zap.Logger

	listener net.Listener
	workers  map[string]*workerConnection

	ShadowData       shared.ShadowData
	HeartbeatSeconds int
}

// NewController creates a new Controller object.
func NewController(logger *zap.Logger, shadowData shared.ShadowData, heartbeat int) *Controller {
	return &Controller{
		Logger:           logger,
		ShadowData:       shadowData,
		HeartbeatSeconds: heartbeat,

		workers: map[string]*workerConnection{},
	}
}

// SetupServer starts listening for connections on the specified port.
//
// This will exit with an error if listening failed.
func (c *Controller) SetupServer(port int) {
	address := fmt.Sprintf("[::]:%d", port)

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

func (c *Controller) handleRegistration(m shared.Message, conn net.Conn) (shared.Message, error) {
	id := conn.RemoteAddr().String()
	ok := c.workers[id].Registered
	if ok {
		err := fmt.Errorf("worker %s is already registered", id)
		return shared.Message{ID: id, Type: shared.MessageError, Timestamp: time.Now(), Message: err.Error()}, err
	}

	c.workers[id].Registered = true
	c.workers[id].Done = make(chan bool)

	return shared.Message{ID: id, Type: shared.MessageRegister, Timestamp: time.Now(), Message: "Registration successful"}, nil
}

func (c *Controller) sendJob(_ shared.Message, conn net.Conn) (shared.Message, error) {
	res := shared.Message{
		Version: shared.MessageVersion, Type: shared.MessageJobDetails, Message: "Cracking details",
		Timestamp: time.Now(),
		Payload: shared.PayloadJobDetails{
			Algorithm:      c.ShadowData.Algorithm,
			Parameters:     c.ShadowData.Parameters,
			Salt:           c.ShadowData.Salt,
			Hash:           c.ShadowData.Hash,
			SearchSpace:    shared.SearchSpace,
			PasswordLength: 3,
		},
	}

	// go c.sendHeartbeat(conn, 5*time.Second)

	return res, nil
}

func (c *Controller) handleJobResults(m shared.Message, conn net.Conn) (shared.Message, error) {
	payload, ok := m.Payload.(shared.PayloadJobResults)
	if !ok {
		return shared.Message{Version: shared.MessageVersion, Type: shared.MessageError, Message: "Bad payload"}, nil
	}

	res := shared.Message{
		Version:   shared.MessageVersion,
		Type:      shared.MessageJobResults,
		Message:   "Received message details",
		Timestamp: time.Now(),
	}

	c.displayJobResults(payload.Password, len(payload.Password) > 0)

	return res, nil
}

func (c *Controller) handleClose(_ shared.Message, conn net.Conn) (shared.Message, error) {
	id := conn.RemoteAddr().String()
	_, ok := c.workers[id]
	if ok {
		delete(c.workers, id)
	}

	message := fmt.Sprintf("Closing connection for %s", id)
	return shared.Message{ID: id, Type: shared.MessageClose, Timestamp: time.Now(), Message: message}, nil
}

func (c *Controller) displayJobResults(result string, cracked bool) {
	if !cracked {
		c.Logger.Info("JOB RESULTS: Failed to crack password", zap.String("password", result))
	} else {
		c.Logger.Info("JOB RESULTS: Cracked password", zap.String("password", result))
	}
}

func (c *Controller) handleHeartbeat(m shared.Message, conn net.Conn) (shared.Message, error) {
	payload, _ := m.Payload.(shared.PayloadHearbeat)

	c.Logger.Info("Heartbeat info",
		zap.Int("delta", payload.DeltaTested),
		zap.Int("activeThreads", payload.ActiveThreads),
	)

	return shared.Message{}, nil
}

func (c *Controller) sendHeartbeat(conn net.Conn, period time.Duration) {
	id := conn.RemoteAddr().String()
	m := shared.Message{
		Version:   shared.MessageVersion,
		ID:        id,
		Type:      shared.MessageHeartbeat,
		Timestamp: time.Now(),
		Message:   "Sending heartbeat",
	}
	worker := c.workers[id]
	ticker := time.NewTicker(period)

	for {
		select {
		case <-worker.Done:
			return
		case <-ticker.C:
			err := worker.Router.Send(m)
			if err != nil {
				ticker.Stop()
				return
			}
		}
	}
}

// HandleConnection manages communication with a single worker for the
// whole entire lifecycle.
func (c *Controller) HandleConnection(conn net.Conn) {
	r := shared.NewRouter(c.Logger, conn)
	r.Handle(shared.MessageRegister, c.handleRegistration)
	r.Handle(shared.MessageJobDetails, c.sendJob)
	r.Handle(shared.MessageJobResults, c.handleJobResults)
	r.Handle(shared.MessageClose, c.handleClose)
	r.Handle(shared.MessageHeartbeat, c.handleHeartbeat)

	c.workers[conn.RemoteAddr().String()] = &workerConnection{
		Registered: false,
		Conn:       conn, Router: r,
	}

	if err := r.Start(); err != nil {
		c.Logger.Error(err.Error())
	}
}
