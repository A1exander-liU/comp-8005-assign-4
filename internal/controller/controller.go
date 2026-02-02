// Package controller contains class for controller
package controller

import (
	"encoding/gob"
	"fmt"
	"net"
	"os"
	"strings"
	"time"

	"github.com/A1exander-liU/comp-8005-assign-1/internal/shared"
	"go.uber.org/zap"
)

type Timing struct {
	Parse, Dispatch, Crack, Return, Total                     time.Time
	ParseDone, DispatchDone, CrackDone, ReturnDone, TotalDone time.Duration
}

// Config holds parameters for controller setup.
type Config struct {
	// Path to the shadowfile
	Shadowfile string

	// User's name to crack
	Username string

	// Port number for the controller to listen on
	Port int
}

// Controller is reponsible for managing worker connections
// for sending and receiving password cracking jobs.
type Controller struct {
	// Public logger for sending log messages
	Logger *zap.Logger

	listener   net.Listener
	ShadowData shared.ShadowData

	Timing Timing
}

// NewController creates a new Controller object.
func NewController(logger *zap.Logger, shadowData shared.ShadowData) *Controller {
	return &Controller{
		Logger:     logger,
		ShadowData: shadowData,
		Timing:     Timing{},
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

func (c *Controller) handleRegistration(conn net.Conn) {
	c.Logger.Info("New worker connected", zap.String("address", conn.RemoteAddr().String()))
}

func (c *Controller) sendRegistrationConfirmation(encoder *gob.Encoder) shared.Message {
	m := shared.Message{
		Version: shared.MessageVersion,
		Type:    shared.MessageRegistrationConfirm,
		Message: "Sending registration confirmation",
	}

	_ = encoder.Encode(m)

	return m
}

func (c *Controller) sendJob(encoder *gob.Encoder) shared.Message {
	m := shared.Message{
		Version: shared.MessageVersion, Type: shared.MessageJobDetails, Message: "Cracking details",
		Data: c.ShadowData,
		PasswordData: shared.PasswordData{
			SearchSpace:    shared.SearchSpace,
			PasswordLength: 3,
		},
	}
	_ = encoder.Encode(m)

	return m
}

func (c *Controller) handleJobResults(m shared.Message) (string, bool) {
	if strings.Contains(m.Message, "failed to crack") {
		return m.Message, false
	}

	return m.Message, true
}

func (c *Controller) displayJobResults(result string, cracked bool) {
	if !cracked {
		c.Logger.Info("Failed to crack password", zap.String("message", result))
	} else {
		c.Logger.Info("Cracked password", zap.String("message", result))
	}
}

func (c *Controller) cleanup() {
	parseTime := float64(c.Timing.ParseDone.Microseconds()) / 1000
	dispatchTime := float64(c.Timing.DispatchDone.Microseconds()) / 1000
	returnTime := float64(c.Timing.ReturnDone.Microseconds()) / 1000

	c.Logger.Info("Timing information",
		zap.String("parse", fmt.Sprintf("%fms", parseTime)),
		zap.String("dispatch", fmt.Sprintf("%fms", dispatchTime)),
		zap.String("crack", fmt.Sprintf("%dms", c.Timing.CrackDone.Milliseconds())),
		zap.String("return", fmt.Sprintf("%fms", returnTime)),
		zap.String("total", fmt.Sprintf("%dms", c.Timing.TotalDone.Milliseconds())),
	)
	_ = c.Logger.Sync()
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
			return
		}

		c.Logger.Info("Received message",
			zap.String("version", m.Version),
			zap.String("message", m.Message),
		)

		var toSend shared.Message

		switch m.Type {
		case shared.MessageRegistration:
			c.Timing.Total = time.Now()
			c.Timing.Dispatch = time.Now()
			c.handleRegistration(conn)
			toSend = c.sendRegistrationConfirmation(encoder)
		case shared.MessageRegistrationConfirm:
			toSend = c.sendJob(encoder)
			c.Timing.DispatchDone = time.Since(c.Timing.Dispatch)
			c.Timing.Crack = time.Now()
			c.Timing.Return = time.Now()
		case shared.MessageJobResults:
			result, cracked := c.handleJobResults(m)
			c.Timing.CrackDone = m.Time
			c.Timing.ReturnDone = time.Since(c.Timing.Return) - m.Time
			c.displayJobResults(result, cracked)
		case shared.MessageConnectionClose:
			toSend = shared.Message{
				Version: shared.MessageVersion,
				Type:    shared.MessageConnectionClose,
				Message: "Confirming connection close",
			}
			_ = encoder.Encode(toSend)
			c.Timing.TotalDone = time.Since(c.Timing.Total)
			c.cleanup()
			return
		}

		c.Logger.Info("Sent message",
			zap.String("version", toSend.Version),
			zap.String("message", toSend.Message),
		)
	}
}
