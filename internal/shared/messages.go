// Package shared contains data types and functions used between the controller and worker
package shared

const (
	MessageVersion string = "1.0.0"
	SearchSpace    string = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789@#%^&*()_+-=.,:;?"
)

type MessageType string

const (
	MessageRegistration        MessageType = "registration.request"
	MessageRegistrationConfirm MessageType = "registration.confirm"
	MessageJobDetails          MessageType = "job.details"
	MessageJobResults          MessageType = "job.results"
	MessageConnectionClose     MessageType = "connection.terminate"
)

type ShadowData struct {
	Username, Algorithm, Parameters, Salt, Hash string
}

type PasswordData struct {
	SearchSpace    string
	PasswordLength int
}

type Message struct {
	Type             MessageType
	Version, Message string
	Data             ShadowData
	PasswordData     PasswordData
}
