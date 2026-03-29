package controller

import (
	"fmt"
	"time"

	"github.com/A1exander-liU/comp-8005-assign-2/internal/shared"
)

// sendJob handles sending job details to a worker, only registered workers can receive job details.
//
// An error will be returned in the message if:
//   - The worker is not registered
func (c *Controller) sendJob(_ shared.Message, id string) (shared.Message, error) {
	if !c.workers[id].Registered {
		err := fmt.Sprintf("worker %s is not registered", id)
		return shared.Message{
				Version:   shared.MessageVersion,
				ID:        id,
				Type:      shared.MessageJobDetails,
				Timestamp: time.Now(),
				Message:   "Job assignment failed",
				Err:       err,
			},
			nil
	}

	timestamp := time.Now()
	if _, ok := c.metric.GetMetric(MetricCrackStart); !ok {
		c.metric.SetMetric(MetricCrackStart, time.Time{})
	}
	if c.crackStart == nil {
		c.crackStart = &timestamp
	}

	assignTS := time.Now()
	chunkID, _ := c.getUnassignedChunk(id)

	c.metric.SetJobMetric(chunkID, JobMetric{
		assignmentStart: assignTS, assignmentEnd: time.Now(),
	})

	c.chunkTimings[chunkID] = &chunkTiming{
		dispatchStart:    timestamp,
		chunkAssignTime:  time.Since(timestamp),
		chunkAssignStart: timestamp,
	}

	res := shared.Message{
		Version: shared.MessageVersion, ID: id, Type: shared.MessageJobDetails, Message: "Job assignment successful",
		Timestamp: timestamp,
		Payload: shared.PayloadJobDetails{
			Algorithm:          c.ShadowData.Algorithm,
			Parameters:         c.ShadowData.Parameters,
			Salt:               c.ShadowData.Salt,
			Hash:               c.ShadowData.Hash,
			ChunkID:            chunkID,
			ChunkStart:         c.chunks[chunkID].start,
			ChunkEnd:           c.chunks[chunkID].end,
			ChunkIndex:         c.chunks[chunkID].index,
			HeartbeatSeconds:   c.Config.HeartbeatSeconds,
			CheckpointAttempts: c.Config.CheckpointAttempts,
		},
	}

	c.LatencyDispatch = timestamp.Sub(c.LatencyDispatchTime)

	return res, nil
}
