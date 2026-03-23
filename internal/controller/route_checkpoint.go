package controller

import (
	"fmt"
	"time"

	"github.com/A1exander-liU/comp-8005-assign-2/internal/shared"
	"go.uber.org/zap"
)

// handleJobCheckpoint updates chunk cracking progress to allow subsequent workers to continue
// from previously recorded progress.
//
// An error will be returned in the message if (the checkpoint will be rejected):
//   - The worker reporting is not currently assigned to the job
func (c *Controller) handleJobCheckpoint(m shared.Message, id string) (shared.Message, error) {
	payload := m.Payload.(shared.PayloadCheckpoint)
	if payload.ChunkID != c.workers[id].ChunkID {
		err := fmt.Sprintf("worker %s is not assigned to chunk %d", id, payload.ChunkID)
		return shared.Message{
				Version:   shared.MessageVersion,
				ID:        id,
				Type:      shared.MessageJobCheckpoint,
				Timestamp: time.Now(),
				Message:   "Job checkpoint rejected",
				Err:       err,
			},
			nil
	}

	c.Logger.Info("Received checkpoint", zap.String("worker", id), zap.Int("chunkID", payload.ChunkID))

	return shared.Message{}, nil
}
