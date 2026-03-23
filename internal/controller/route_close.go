package controller

import (
	"fmt"
	"time"

	"github.com/A1exander-liU/comp-8005-assign-2/internal/shared"
)

// handleClose performs cleanup logic after a worker requests to close their connection.
//
// This will delete their entry as a worker and also reclaim any ongoing work they have.
func (c *Controller) handleClose(_ shared.Message, id string) (shared.Message, error) {
	if worker, ok := c.workers[id]; ok {
		c.revokeJob(id, worker.ChunkID)
		delete(c.workers, id)
	}

	message := fmt.Sprintf("Closing connection for %s", id)
	return shared.Message{
			Version:   shared.MessageVersion,
			ID:        id,
			Type:      shared.MessageClose,
			Timestamp: time.Now(),
			Message:   message,
		},
		nil
}
