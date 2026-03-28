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
		err := fmt.Sprintf("worker %s was not already connected", payload.ID)
		return shared.Message{
				Version:   shared.MessageVersion,
				Type:      shared.MessageReconnect,
				Timestamp: time.Now(),
				Message:   "Reconnection failed",
				Err:       err,
			},
			nil
	}

	chunkID := c.workers[payload.ID].ChunkID
	oldChannel := c.workers[payload.ID].reconnectionChan

	c.workers[payload.ID] = c.workers[id]
	c.workers[payload.ID].Registered = true
	c.workers[payload.ID].ChunkID = chunkID
	c.workers[payload.ID].reconnectionChan = oldChannel
	c.workers[payload.ID].Router.ID = payload.ID
	delete(c.workers, id)

	c.workers[payload.ID].reconnectionChan <- true

	return shared.Message{
			Version:   shared.MessageVersion,
			Type:      shared.MessageReconnect,
			Timestamp: time.Now(),
			Message:   "Reconnection successful",
		},
		nil
}
