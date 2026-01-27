// Package worker contains class for worker
package worker

import (
	"encoding/gob"
	"fmt"
	"net"
	"os"

	"github.com/A1exander-liU/comp-8005-assign-1/internal/shared"
	"go.uber.org/zap"
)

// Config holds parameters for worker setup:
//
// - ControllerIP is the IP address of the controller to connect to
//
// - ControllerPort is the port number of the controller to connect to
type Config struct {
	ControllerIP   string
	ControllerPort int
}

// Worker is reponsible for receiving password cracking jobs from
// the controller and sending the results back.
type Worker struct {
	Logger *zap.Logger
	conn   net.Conn
}

func NewWorker(logger *zap.Logger) *Worker {
	return &Worker{
		Logger: logger,
	}
}

func (w *Worker) SetupServer(config Config) {
	address := net.JoinHostPort(config.ControllerIP, string(config.ControllerPort))

	conn, err := net.Dial("tcp", address)
	if err != nil {
		fmt.Printf("Failed to connected to %s: %s\n", address, err)
		os.Exit(1)
	}

	w.Logger.Info(fmt.Sprintf("Connected to controller at %s", address))
	w.conn = conn
}

func (w *Worker) sendRegistration(encoder *gob.Encoder) {
	m := shared.Message{
		Version: shared.MessageVersion,
		Type:    shared.MessageRegistration,
		Message: "Sending registration request",
	}

	_ = encoder.Encode(m)
}

func (w *Worker) HandleConnection() {
	encoder := gob.NewEncoder(w.conn)
	decoder := gob.NewDecoder(w.conn)

	w.sendRegistration(encoder)
	for {
		var m shared.Message

		if err := decoder.Decode(&m); err != nil {
			w.Logger.Error("Failed to decode incoming message", zap.Error(err))
			continue
		}

		w.Logger.Info("Received message",
			zap.String("version", m.Version),
			zap.String("message", m.Message),
		)
	}
}
