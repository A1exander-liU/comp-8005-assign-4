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

	// Worker id, IP:PORT format
	ID string

	conn    net.Conn
	encoder *gob.Encoder
	decoder *gob.Decoder

	router *shared.Router
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

// Get sends and receives a message with the controller.
func (w *Worker) get(m shared.Message) (shared.Message, error) {
	if err := w.encoder.Encode(m); err != nil {
		return shared.Message{}, err
	}
	w.Logger.Info("Sent", zap.String("message", m.Message), zap.Time("timestamp", m.Timestamp))

	var message shared.Message
	if err := w.decoder.Decode(&message); err != nil {
		return shared.Message{}, err
	}
	w.Logger.Info("Received", zap.String("message", message.Message), zap.Time("timestamp", m.Timestamp))

	return message, nil
}

func (w *Worker) sendRegistration() (shared.Message, error) {
	m := shared.Message{
		Version:   shared.MessageVersion,
		ID:        w.ID,
		Type:      shared.MessageRegister,
		Timestamp: time.Now(),
		Message:   "Sending registration request",
	}

	return w.get(m)
}

func (w *Worker) sendJobRequest() (shared.Message, error) {
	m := shared.Message{
		Version:   shared.MessageVersion,
		ID:        w.ID,
		Type:      shared.MessageJobDetails,
		Timestamp: time.Now(),
		Message:   "Sending request for job",
	}

	return w.get(m)
}

func (w *Worker) sendJobResults(result string, err error, crackTime time.Duration) (shared.Message, error) {
	m := shared.Message{
		Version:   shared.MessageVersion,
		ID:        w.ID,
		Type:      shared.MessageJobResults,
		Timestamp: time.Now(),
	}

	jobResults := shared.PayloadJobResults{}
	if err != nil {
		w.Logger.Info("Failed to crack password")

		m.Message = err.Error()
		jobResults.Password = ""
		jobResults.Time = crackTime
	} else {
		w.Logger.Info("Sucessfully cracked password")

		m.Message = "Password cracked"
		jobResults.Password = result
		jobResults.Time = crackTime
	}
	m.Payload = jobResults

	return w.get(m)
}

func (w *Worker) sendClose() (shared.Message, error) {
	m := shared.Message{
		Version:   shared.MessageVersion,
		ID:        w.ID,
		Type:      shared.MessageClose,
		Message:   "Sending close request",
		Timestamp: time.Now(),
	}

	res, err := w.get(m)
	return res, err
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

func (w *Worker) handleHeartbeat(m shared.Message, conn net.Conn) (shared.Message, error) {
	return shared.Message{}, nil
}

func (w *Worker) cleanup() {
	_ = w.Logger.Sync()
	_ = w.conn.Close()
}

// HandleConnection handles worker lifecycle of sending and receiving to and from the controller.
func (w *Worker) HandleConnection() {
	// w.router = shared.NewRouter(w.Logger, w.conn)
	// w.router.Handle(shared.MessageHeartbeat, w.handleHeartbeat)
	// go w.router.Start()

	_, err := w.sendRegistration()
	if err != nil {
		w.cleanup()
		return
	}

	res, err := w.sendJobRequest()
	if err != nil {
		w.cleanup()
		return
	}

	payload, _ := res.Payload.(shared.PayloadJobDetails)
	start := time.Now()
	w.Logger.Info("Cracking...")
	result, err := w.handleJob(payload)
	end := time.Since(start)

	_, err = w.sendJobResults(result, err, end)
	if err != nil {
		w.cleanup()
		return
	}

	_, err = w.sendClose()
	if err != nil {
		return
	}
}
