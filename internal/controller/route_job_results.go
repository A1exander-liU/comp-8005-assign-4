package controller

import (
	"errors"
	"fmt"
	"net"
	"time"

	"github.com/A1exander-liU/comp-8005-assign-2/internal/shared"
)

func (c *Controller) handleJobResults(m shared.Message, conn net.Conn) (shared.Message, error) {
	id := conn.RemoteAddr().String()
	worker, ok := c.workers[id]

	// check if worker is registered
	if !ok {
		err := fmt.Errorf("worker %s not registered", id)
		return shared.Message{
				Version:   shared.MessageVersion,
				Type:      shared.MessageError,
				Timestamp: time.Now(),
				Message:   err.Error(),
			},
			nil
	}

	payload, ok := m.Payload.(shared.PayloadJobResults)
	if !ok {
		return shared.Message{Version: shared.MessageVersion, Type: shared.MessageError, Message: "Bad payload"}, nil
	}

	// check if worker is assigned to the job
	if worker.ChunkID != payload.ChunkID {
		err := fmt.Errorf("worker %s is not assigned to chunk %d", id, payload.ChunkID)
		return shared.Message{
				Version:   shared.MessageVersion,
				Type:      shared.MessageError,
				Timestamp: time.Now(),
				Message:   err.Error(),
			},
			nil
	}

	timestamp := time.Now()

	c.chunkTimings[payload.ChunkID].crackTime = payload.CrackTime
	c.chunkTimings[payload.ChunkID].dispatchTime = payload.DispatchTime.Abs()
	c.chunkTimings[payload.ChunkID].returnTime = time.Since(m.Timestamp).Abs()

	var err error
	var done bool
	if payload.Err != "" {
		err = errors.New(payload.Err)
		done = false
	} else {
		err = nil
		done = true
	}

	c.chunks[payload.ChunkID].status = ChunkCompleted
	c.workers[conn.RemoteAddr().String()].ChunkID = -1

	res := shared.Message{
		Version:   shared.MessageVersion,
		Type:      shared.MessageJobResults,
		Message:   "Received message details",
		Timestamp: time.Now(),
		Payload:   shared.PayloadJobResultsResp{Done: done},
	}

	c.LatencyCrack = payload.CrackTime
	c.LatencyReturn = m.Timestamp.Sub(timestamp)

	c.displayJobResults(payload.Password, err, payload.ChunkID, timestamp)

	if err == nil {
		c.sendStop()
	}

	return res, nil
}
