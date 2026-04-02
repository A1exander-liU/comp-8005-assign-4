package controller

import (
	"fmt"
	"time"

	"github.com/A1exander-liU/comp-8005-assign-4/internal/shared"
)

func (c *Controller) handleReconnect(m shared.Message, id string) (shared.Message, error) {
	payload := m.Payload.(shared.PayloadReconnect)
	_, ok := c.workers[payload.ID]

	if !ok {
		err := fmt.Sprintf("worker %s was not already connected", payload.ID)
		return shared.Message{
				Version:   shared.MessageVersion,
				Type:      shared.MessageReconnect,
				Timestamp: time.Now().UTC(),
				Message:   "Reconnection failed",
				Err:       err,
			},
			nil
	}

	chunkID := c.workers[payload.ID].ChunkID
	reconnectionC := c.workers[payload.ID].reconnectionC

	c.workers[payload.ID] = c.workers[id]

	c.workers[payload.ID].Registered = true
	c.workers[payload.ID].Connected = true
	c.workers[payload.ID].ChunkID = chunkID
	c.workers[payload.ID].Router.ID = payload.ID
	c.workers[payload.ID].reconnectionC = reconnectionC
	delete(c.workers, id)

	c.workers[payload.ID].reconnectionC <- true

	return shared.Message{
			Version:   shared.MessageVersion,
			Type:      shared.MessageReconnect,
			Timestamp: time.Now().UTC(),
			Message:   "Reconnection successful",
			Payload:   shared.PayloadReconnectResp{ChunkID: chunkID},
		},
		nil
}
