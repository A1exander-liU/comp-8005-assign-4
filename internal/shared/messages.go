// Package shared contains data types and functions used between the controller and worker
package shared

type MessageType string

const (
	MessageRegistration         MessageType = "registration.request"
	MessageRegistrationConfirma MessageType = "registration.confirm"
	MessageJobDetails           MessageType = "job.details"
	MessageJobResults           MessageType = "job.results"
	MessageConnectionClose      MessageType = "connection.terminate"
)

type ShadowData struct {
	Username, Algorithm, Parameters, Salt, Hash string
}

type Message struct {
	Version, Type, Result string
	Message               MessageType
	Data                  ShadowData
}
