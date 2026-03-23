package controller

import (
	"fmt"
	"time"

	"github.com/A1exander-liU/comp-8005-assign-2/internal/shared"
)

func (c *Controller) handleReconnect(m shared.Message, id string) (shared.Message, error) {
	payload := m.Payload.(shared.PayloadReconnect)
	_, ok := c.workers[payload.ID]

	if !ok {
		err := fmt.Sprintf("worker %s was not already connected", id)
		return shared.Message{
				Version:   shared.MessageVersion,
				Type:      shared.MessageReconnect,
				Timestamp: time.Now(),
				Message:   "Reconnection failed",
				Err:       err,
			},
			nil
	}

	c.workers[payload.ID] = c.workers[id]
	// delete new entry
	delete(c.workers, id)

	return shared.Message{
			Version:   shared.MessageVersion,
			Type:      shared.MessageReconnect,
			Timestamp: time.Now(),
			Message:   "Reconnection successful",
		},
		nil
}
