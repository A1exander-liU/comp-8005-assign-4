package worker

import (
	"github.com/A1exander-liU/comp-8005-assign-2/internal/shared"
	"go.uber.org/zap"
)

func (w *Worker) routeJobResults(m shared.Message, s string) (shared.Message, error) {
	if m.Err != "" {
		w.Logger.Warn("Job results rejected", zap.String("error", m.Err))
	}

	payload := m.Payload.(shared.PayloadJobResultsResp)

	// password was found, exit
	if payload.Done {
		return shared.Message{Type: shared.MessageClose}, nil
	}

	return shared.Message{Type: shared.MessageJobDetails}, nil
}
