// Package worker contains class for worker
package worker

import (
	"encoding/gob"
	"fmt"
	"io"
	"net"
	"os"
	"strconv"
	"strings"
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
}

// Worker is reponsible for receiving password cracking jobs from
// the controller and sending the results back.
type Worker struct {
	// Public logger for sending log messages
	Logger *zap.Logger

	// Worker id, IP:PORT format
	ID string

	conn    net.Conn
	encoder *gob.Encoder
	decoder *gob.Decoder
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
	w.ID = conn.LocalAddr().String()
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

func (w *Worker) sendJobResults(result string, err error, crackTime time.Duration) error {
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
		jobResults.Time = crackTime
	} else {
		w.Logger.Info("Sucessfully cracked password")

		m.Message = "Sending job results: Password cracked"
		jobResults.Password = result
		jobResults.Time = crackTime
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
	m := shared.Message{
		Version:   shared.MessageVersion,
		ID:        w.ID,
		Type:      shared.MessageHeartbeat,
		Message:   "Sending heartbeat",
		Timestamp: time.Now(),
	}

	return w.send(m)
}

func (w *Worker) handleJob(payload shared.PayloadJobDetails) (string, error) {
	decoder, _ := crypt.NewDecoderAll()

	sections := make([]string, 0)
	if payload.Algorithm != "" {
		sections = append(sections, fmt.Sprintf("$%s", payload.Algorithm))
	}
	if payload.Parameters != "" {
		sections = append(sections, payload.Parameters)
	}
	if payload.Salt != "" {
		sections = append(sections, payload.Salt)
	}
	if payload.Hash != "" {
		sections = append(sections, payload.Hash)
	}

	fullHash := strings.Join(sections, "$")

	return shared.CrackPassword(decoder, fullHash, payload.SearchSpace, payload.PasswordLength)
}

func (w *Worker) cleanup() {
	_ = w.Logger.Sync()
	_ = w.conn.Close()
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
			break outer
		}
		if err != nil {
			w.Logger.Error("Failed to decode", zap.Error(err))
			continue
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
			w.Logger.Info("Received job details", zap.String("algorithm", payload.Algorithm))

			// Start cracking here
			w.Logger.Info("Cracking password...")
			start := time.Now()
			result, err := w.handleJob(payload)
			end := time.Since(start)

			err = w.sendJobResults(result, err, end)
			if err != nil {
				break outer
			}
			continue

		case shared.MessageJobResults:
			err := w.sendClose()
			if err != nil {
				break outer
			}

		case shared.MessageHeartbeat:
			err := w.handleHeartbeat()
			if err != nil {
				break outer
			}

		case shared.MessageError:
			break outer

		case shared.MessageClose:
			break outer
		}
	}

	w.cleanup()
}
