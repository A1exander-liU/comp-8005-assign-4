package controller

import (
	"fmt"
	"net"
	"time"

	"github.com/A1exander-liU/comp-8005-assign-2/internal/shared"
)

// handleRegistration updates worker connection information by setting the
// registered flag.
//
// An `MessageError` will be returned if worker already registered.
func (c *Controller) handleRegistration(m shared.Message, conn net.Conn) (shared.Message, error) {
	id := conn.RemoteAddr().String()
	ok := c.workers[id].Registered
	if ok {
		err := fmt.Errorf("worker %s is already registered", id)
		return shared.Message{ID: id, Type: shared.MessageError, Timestamp: time.Now(), Message: err.Error()}, err
	}

	c.workers[id].Registered = true

	c.LatencyDispatchTime = time.Now()

	return shared.Message{
			Version:   shared.MessageVersion,
			ID:        id,
			Type:      shared.MessageRegister,
			Timestamp: time.Now(),
			Message:   "Registration successful",
			Payload: shared.PayloadRegisterResp{
				HeartbeatSeconds:   c.Config.HeartbeatSeconds,
				CheckpointAttempts: c.Config.CheckpointAttempts,
			},
		},
		nil
}
