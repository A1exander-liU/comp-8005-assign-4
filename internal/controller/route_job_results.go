package controller

import (
	"errors"
	"fmt"
	"time"

	"github.com/A1exander-liU/comp-8005-assign-4/internal/shared"
)

func (c *Controller) handleJobResults(m shared.Message, id string) (shared.Message, error) {
	worker, ok := c.workers[id]

	// check if worker is registered
	if !ok {
		err := fmt.Sprintf("worker %s not registered", id)
		return shared.Message{
				Version:   shared.MessageVersion,
				ID:        id,
				Type:      shared.MessageJobResults,
				Timestamp: time.Now(),
				Message:   "Job results rejected",
				Err:       err,
			},
			nil
	}

	payload := m.Payload.(shared.PayloadJobResults)

	// check if worker is assigned to the job
	if worker.ChunkID != payload.ChunkID {
		err := fmt.Sprintf("worker %s is not assigned to chunk %d", id, payload.ChunkID)
		return shared.Message{
				Version:   shared.MessageVersion,
				ID:        id,
				Type:      shared.MessageJobResults,
				Timestamp: time.Now(),
				Message:   "Job results rejected",
				Err:       err,
			},
			nil
	}

	timestamp := time.Now()

	c.metric.SetJobMetric(payload.ChunkID, JobMetric{
		dispatchTime: payload.DispatchTime,
		crackTime:    payload.CrackTime,
		returnStart:  m.Timestamp, returnEnd: timestamp,
	})

	var err error
	var done bool
	if payload.Err != "" {
		err = errors.New(payload.Err)
		done = false
	} else {
		err = nil
		done = true
		c.metric.SetMetric(MetricCrackEnd, time.Time{})
	}

	c.chunks[payload.ChunkID].status = ChunkCompleted
	c.workers[id].ChunkID = -1

	res := shared.Message{
		Version:   shared.MessageVersion,
		ID:        id,
		Type:      shared.MessageJobResults,
		Timestamp: time.Now(),
		Message:   "Job results accepted",
		Payload:   shared.PayloadJobResultsResp{Done: done},
	}

	// c.displayJobResults(payload.Password, err, payload.ChunkID, timestamp)
	c.printJobResults(payload.Password, err, payload.ChunkID)

	if err == nil {
		c.printFinalResults(payload.Password, err)
		c.sendStop()
	}

	return res, nil
}
