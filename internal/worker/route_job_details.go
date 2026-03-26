package worker

import (
	"github.com/A1exander-liU/comp-8005-assign-2/internal/shared"
	"go.uber.org/zap"
)

func (w *Worker) routeJobDetails(m shared.Message, id string) (shared.Message, error) {
	if m.Err != "" {
		w.Logger.Warn("Failed to receive job", zap.String("error", m.Err))
		return shared.Message{Type: shared.MessageClose}, nil
	}

	payload := m.Payload.(shared.PayloadJobDetails)

	go w.handleJobV1(payload)

	return shared.Message{}, nil
}
