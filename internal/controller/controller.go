// Package controller contains class for controller
package controller

import (
	"encoding/gob"
	"fmt"
	"net"
	"os"

	"github.com/A1exander-liU/comp-8005-assign-1/internal/shared"
	"go.uber.org/zap"
)

// Config holds parameters for controller setup:
//
// - Shadowfile is the path to the shadowfile
//
// - Username is the username to crack password of
//
// - Port is the port number the controller should listen on
type Config struct {
	Shadowfile, Username string
	Port                 int
}

// Controller is reponsible for managing worker connections
// for sending and receiving password cracking jobs.
type Controller struct {
	Logger   *zap.Logger
	listener net.Listener
}

// NewController creates a new Controller object with defaults:
//
// - A
//
// - B
func NewController(logger *zap.Logger) *Controller {
	return &Controller{
		Logger: logger,
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

// HandleConnection manages communication with a single worker for the
// whole entire lifecycle.
func (c *Controller) HandleConnection(conn net.Conn) {
	encoder := gob.NewEncoder(conn)
	decoder := gob.NewDecoder(conn)

	for {
		var m shared.Message

		if err := decoder.Decode(&m); err != nil {
			c.Logger.Error("Failed to decode incoming message", zap.Error(err))
			continue
		}

		c.Logger.Info("Received message",
			zap.String("version", m.Version),
			zap.String("message", m.Message),
		)

		_ = encoder.Encode(shared.Message{
			Version: "1", Message: fmt.Sprintf("Received message: %s", m.Message),
		})

	}
}
