package worker

import (
	"github.com/A1exander-liU/comp-8005-assign-2/internal/shared"
	"go.uber.org/zap"
)

func (w *Worker) routeReconnect(m shared.Message, s string) (shared.Message, error) {
	if m.Err != "" {
		w.Logger.Warn("Reconnection failed, registering instead...", zap.String("error", m.Err))
		return shared.Message{Type: shared.MessageRegister}, nil
	}

	w.lastAttemptsSent = w.getTotalAttempts()

	w.Logger.Info("Reconnection successful")

	go w.handleJobV1(w.state.Payload)

	return shared.Message{}, nil
}
