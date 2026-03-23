package controller

import (
	"fmt"
	"net"
	"time"

	"github.com/A1exander-liU/comp-8005-assign-2/internal/shared"
	"go.uber.org/zap"
)

func (c *Controller) handleJobCheckpoint(m shared.Message, conn net.Conn) (shared.Message, error) {
	id := conn.RemoteAddr().String()

	payload, ok := m.Payload.(shared.PayloadCheckpoint)
	if !ok {
		err := fmt.Errorf("expected PayloadCheckpoint")
		return shared.Message{
				Version:   shared.MessageVersion,
				ID:        id,
				Type:      shared.MessageError,
				Timestamp: time.Now(),
				Message:   err.Error(),
			},
			err
	}

	c.Logger.Info("received checkpoint", zap.String("worker", id), zap.Int("chunkID", payload.ChunkID))

	return shared.Message{}, nil
}
