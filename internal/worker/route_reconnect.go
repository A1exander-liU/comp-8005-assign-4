package worker

import (
	"time"

	"github.com/A1exander-liU/comp-8005-assign-4/internal/shared"
	"go.uber.org/zap"
)

func (w *Worker) routeReconnect(m shared.Message, s string) (shared.Message, error) {
	if m.Err != "" {
		w.Logger.Warn("Reconnection failed, registering instead...", zap.String("error", m.Err))
		return shared.Message{Type: shared.MessageRegister, Timestamp: time.Now()}, nil
	}

	w.lastAttemptsSent = w.getTotalAttempts()

	w.Logger.Info("Reconnection successful")

	go w.handleJobV1(w.state.Payload, time.Time{}, time.Now())

	return shared.Message{}, nil
}
