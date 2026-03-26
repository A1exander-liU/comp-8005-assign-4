package worker

import (
	"github.com/A1exander-liU/comp-8005-assign-2/internal/shared"
	"go.uber.org/zap"
)

func (w *Worker) routeRegister(m shared.Message, _ string) (shared.Message, error) {
	if m.Err != "" {
		w.Logger.Warn("Registration failed", zap.String("error", m.Err))
		return shared.Message{Type: shared.MessageClose}, nil
	}

	payload := m.Payload.(shared.PayloadRegisterResp)
	w.state = InitialStateWithID(payload.ID)
	w.lastAttemptsSent = 0

	_ = SaveState(StateFileLocation, w.state)

	w.Logger.Info("Created initial state")

	return shared.Message{Type: shared.MessageJobDetails}, nil
}
