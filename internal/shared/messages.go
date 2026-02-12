// Package shared contains data types and functions used between the controller and worker.
package shared

import (
	"encoding/gob"
	"time"
)

const (
	MessageVersion string = "1.0.0"
	SearchSpace    string = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789@#%^&*()_+-=.,:;?"
)

// MessageType communicates different requests for controllers and workers to respond to.
type MessageType string

const (
	MessageJobDetails MessageType = "job.details"
	MessageJobResults MessageType = "job.results"
	MessageRegister   MessageType = "connection.register"
	MessageHeartbeat  MessageType = "connection.heartbeat"
	MessageError      MessageType = "connection.error"
	MessageClose      MessageType = "connection.close"
)

type Message struct {
	Version   string
	ID        string
	Type      MessageType
	Timestamp time.Time
	Message   string
	Payload   any
}
type ShadowData struct {
	Username, Algorithm, Parameters, Salt, Hash string
}

type PayloadJobDetails struct {
	// Password hash details
	Username, Algorithm, Parameters, Salt, Hash string
	// A string of individual for the worker to generate candidate passwords from
	SearchSpace    string
	PasswordLength int
}

type PayloadJobResults struct {
	// The cracked password, will be empty if cracking failed
	Password string
	Time     time.Duration
}

type PayloadHearbeat struct {
	// Number of password candidates tested since last heartbeat
	DeltaTested int

	// Number of threads currently utilised by the worker for password cracking
	ActiveThreads int
}

// RegisterMessages registers the message structs to enable decoding of any types.
//
// Should be called before attempting to send messages.
func RegisterMessages() {
	gob.Register(PayloadJobResults{})
	gob.Register(PayloadJobDetails{})
	gob.Register(PayloadHearbeat{})
}
