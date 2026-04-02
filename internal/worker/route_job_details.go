package worker

import (
	"time"

	"github.com/A1exander-liU/comp-8005-assign-4/internal/shared"
	"go.uber.org/zap"
)

func (w *Worker) routeJobDetails(m shared.Message, id string) (shared.Message, error) {
	if m.Err != "" {
		w.Logger.Warn("Failed to receive job", zap.String("error", m.Err))
		return shared.Message{Type: shared.MessageClose, Timestamp: time.Now().UTC()}, nil
	}

	payload := m.Payload.(shared.PayloadJobDetails)
	w.Logger.Info("Received job", zap.Int("chunkID", payload.ChunkID))
	w.state.Payload = payload
	w.state.PasswordIndex = int(payload.ChunkIndex)
	w.lastAttemptsSent = 0

	go w.handleJobV1(payload, m.Timestamp, time.Now().UTC())

	return shared.Message{}, nil
}
