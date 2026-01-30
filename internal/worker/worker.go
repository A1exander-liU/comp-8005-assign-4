// Package worker contains class for worker
package worker

import (
	"encoding/gob"
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/A1exander-liU/comp-8005-assign-1/internal/shared"
	"github.com/go-crypt/crypt"
	"go.uber.org/zap"
)

// Config holds parameters for worker setup:
type Config struct {
	// IP address of the controller to connect to
	ControllerIP string

	// Port number of the controller to connect to
	ControllerPort int
}

// Worker is reponsible for receiving password cracking jobs from
// the controller and sending the results back.
type Worker struct {
	// Public logger for sending log messages
	Logger *zap.Logger

	conn net.Conn
}

// NewWorker creates a new worker with the provided logger instance.
func NewWorker(logger *zap.Logger) *Worker {
	return &Worker{
		Logger: logger,
	}
}

// SetupServer creates a connection with the controller.
//
// This will exit with an error if it failed to connect.
func (w *Worker) SetupServer(config Config) {
	address := net.JoinHostPort(config.ControllerIP, strconv.Itoa(config.ControllerPort))

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

func (w *Worker) handleJob(shadowData shared.ShadowData, passwordData shared.PasswordData) (string, error) {
	decoder, _ := crypt.NewDecoderAll()

	sections := make([]string, 0)
	if shadowData.Algorithm != "" {
		sections = append(sections, fmt.Sprintf("$%s", shadowData.Algorithm))
	}
	if shadowData.Parameters != "" {
		sections = append(sections, shadowData.Parameters)
	}
	if shadowData.Salt != "" {
		sections = append(sections, shadowData.Salt)
	}
	if shadowData.Hash != "" {
		sections = append(sections, shadowData.Hash)
	}

	fullHash := strings.Join(sections, "$")

	return shared.CrackPassword(decoder, fullHash, passwordData.SearchSpace, passwordData.PasswordLength)
}

func (w *Worker) sendJobResults(encoder *gob.Encoder, result string, err error, crackTime time.Duration) shared.Message {
	toSend := shared.Message{
		Version: shared.MessageVersion,
		Type:    shared.MessageJobResults,
		Time:    crackTime,
	}

	if err != nil {
		w.Logger.Info("Failed to crack password")
		toSend.Message = err.Error()
	} else {
		w.Logger.Info("Sucessfully cracked password")
		toSend.Message = result
	}

	_ = encoder.Encode(toSend)

	return toSend
}

func (w *Worker) sendTermination(encoder *gob.Encoder) {
	m := shared.Message{
		Version: shared.MessageVersion,
		Type:    shared.MessageConnectionClose,
		Message: "Sending termination request",
	}

	_ = encoder.Encode(m)
}

func (w *Worker) cleanup() {
	_ = w.Logger.Sync()
	_ = w.conn.Close()
}

// TODO: refactor message handling and format
// need also way to time the code (use channels or just a time.now between segments)

// HandleConnection handles worker lifecycle of sending and receiving to and from the controller.
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

		var toSend shared.Message

		switch m.Type {
		case shared.MessageRegistrationConfirm:
			toSend = shared.Message{Version: shared.MessageVersion, Type: shared.MessageRegistrationConfirm, Message: "Sucessfully received confirmation message"}
			_ = encoder.Encode(toSend)
		case shared.MessageJobDetails:
			w.Logger.Info("Start cracking password")
			start := time.Now()
			result, err := w.handleJob(m.Data, m.PasswordData)
			crackTime := time.Since(start)
			toSend = w.sendJobResults(encoder, result, err, crackTime)
			w.sendTermination(encoder)
		case shared.MessageConnectionClose:
			w.Logger.Info("Closing connection")
			w.cleanup()
			return
		}

		w.Logger.Info("Sent message",
			zap.String("version", toSend.Version),
			zap.String("message", toSend.Message),
		)
	}
}
