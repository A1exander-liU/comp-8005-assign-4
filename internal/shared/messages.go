// Package shared contains data types and functions used between the controller and worker.
package shared

const (
	MessageVersion string = "1.0.0"
	SearchSpace    string = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789@#%^&*()_+-=.,:;?"
)

// MessageType communicates different requests for controllers and workers to respond to.
type MessageType string

const (
	MessageRegistration        MessageType = "registration.request"
	MessageRegistrationConfirm MessageType = "registration.confirm"
	MessageJobDetails          MessageType = "job.details"
	MessageJobResults          MessageType = "job.results"
	MessageConnectionClose     MessageType = "connection.terminate"
)

// ShadowData contains extracted information of a single user and
// password in the shadowfile.
type ShadowData struct {
	// user's name
	Username string

	// Algorithm used to generate the password hash
	Algorithm string

	// Optional parameters to supply to the hashing function
	Parameters string

	// Salt to supply to the hashing function
	Salt string

	// The hash of the password
	Hash string
}

// PasswordData contains cracking details for the worker.
type PasswordData struct {
	// The entire character set the worker should try when cracking
	SearchSpace string

	// The length of the password
	PasswordLength int
}

// Message is the communication protocol between the controller and worker.
type Message struct {
	// Type of communication the controller or worker wants to send
	Type MessageType

	// Protocol version to use (in semantic versioning scheme)
	Version string

	// An additional message to send
	Message string

	// Extracted information from shadowfile for whose password to crack
	Data ShadowData

	// Password cracking details for the worker
	PasswordData PasswordData
}
