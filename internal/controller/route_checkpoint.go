package controller

import (
	"time"

	"github.com/A1exander-liU/comp-8005-assign-4/internal/shared"
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
		c.Logger.Info("Checkpoint rejected, not assigned to chunk", zap.String("workerID", id), zap.Int("chunkID", payload.ChunkID))
	} else {
		c.Logger.Info("Checkpoint accepted", zap.String("workerID", id), zap.Int("chunkID", payload.ChunkID), zap.Uint64("chunkIndex", payload.ChunkIndex))
		c.chunks[payload.ChunkID].index = payload.ChunkIndex
	}

	c.metric.AddCheckpointTiming(m.Timestamp, time.Now().UTC())

	return shared.Message{}, nil
}
