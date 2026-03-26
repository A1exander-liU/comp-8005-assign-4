package controller

import (
	"fmt"
	"time"

	"github.com/A1exander-liU/comp-8005-assign-2/internal/shared"
)

// handleRegistration registers a worker by updating their worker connection information by setting the
// registered flag. The response will contain the heartbeat and checkpoint interval.
//
// An error will be in the returned message if:
//   - The worker has already been registered
func (c *Controller) handleRegistration(m shared.Message, id string) (shared.Message, error) {
	ok := c.workers[id].Registered

	if ok {
		err := fmt.Sprintf("worker %s is already registered", id)
		return shared.Message{
				Version:   shared.MessageVersion,
				ID:        id,
				Type:      shared.MessageRegister,
				Timestamp: time.Now(),
				Message:   "Registration failed",
				Err:       err,
			},
			nil
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
				ID: id,
			},
		},
		nil
}
