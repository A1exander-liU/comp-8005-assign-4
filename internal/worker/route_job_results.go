package worker

import (
	"time"

	"github.com/A1exander-liU/comp-8005-assign-4/internal/shared"
	"go.uber.org/zap"
)

func (w *Worker) routeJobResults(m shared.Message, s string) (shared.Message, error) {
	if m.Err != "" {
		w.Logger.Warn("Job results rejected", zap.String("error", m.Err))
	} else {
		payload := m.Payload.(shared.PayloadJobResultsResp)

		// password was found, exit
		if payload.Done {
			return shared.Message{Type: shared.MessageClose, Timestamp: time.Now().UTC()}, nil
		}
	}

	return shared.Message{Type: shared.MessageJobDetails, Timestamp: time.Now().UTC()}, nil
}
