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
	MessageJobDetails    MessageType = "job.details"
	MessageJobCheckpoint MessageType = "job.checkpoint"
	MessageJobResults    MessageType = "job.results"
	MessageRegister      MessageType = "connection.register"
	MessageHeartbeat     MessageType = "connection.heartbeat"
	MessageClose         MessageType = "connection.close"
)

// Message the data protocol between the controller and worker.
type Message struct {
	// Protcol version
	Version string

	// Worker id (this a address port string)
	ID string

	// Indicate the communication, `Payload` will change according to this
	Type MessageType

	// Time the message was sent
	Timestamp time.Time

	// Descriptive text
	Message string

	// Indicates if the request failed to successfully complete.
	// This is meant to be checked when receiving a response.
	Err error

	// Type indicates the struct used.
	//
	// Accessing the payload can be done with:
	//
	// 	payload, ok := message.Payload.(PayloadHearbeat)
	//
	// where PayloadHearbeat would be desired type of the payload.
	Payload any
}

type ShadowData struct {
	Username, Algorithm, Parameters, Salt, Hash string
}

// PayloadRegisterResp contains configuration values to be sent
// to a worker.
type PayloadRegisterResp struct {
	HeartbeatSeconds, CheckpointAttempts int
}

type PayloadJobDetails struct {
	// Password hash details
	Username, Algorithm, Parameters, Salt, Hash string

	ChunkID              int
	ChunkStart, ChunkEnd uint64

	HeartbeatSeconds, CheckpointAttempts int
}

type PayloadJobResults struct {
	// The cracked password, will be empty if cracking failed
	Password                string
	CrackTime, DispatchTime time.Duration
	Err                     string

	ChunkID int
}

type PayloadJobResultsResp struct {
	// Done searching for passwords, indicating to workers the password was found
	Done bool
}

type PayloadHearbeat struct {
	// Number of password candidates tested in total so far
	TotalTested int

	// Number of password candidates tested since last heartbeat
	DeltaTested int

	// Number of threads currently utilised by the worker for password cracking
	ActiveThreads int
}

type PayloadCheckpoint struct {
	ChunkID int

	// Progress made since last checkpoint
	// Progress is an array of length two arrays
	// Each array is for each thread which shows start and end indices of attempted passwords
	// [ [ start, end ], ... ]
	CurrentProgress [][]int

	// Number of attempted passwords
	CurrentTested int
}

// RegisterMessages registers the message structs to enable decoding of any types.
//
// Should be called before attempting to send messages.
func RegisterMessages() {
	gob.Register(PayloadRegisterResp{})
	gob.Register(PayloadJobResults{})
	gob.Register(PayloadJobResultsResp{})
	gob.Register(PayloadJobDetails{})
	gob.Register(PayloadHearbeat{})
	gob.Register(PayloadCheckpoint{})
}
