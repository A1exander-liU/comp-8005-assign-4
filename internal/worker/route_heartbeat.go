package worker

import (
	"time"

	"github.com/A1exander-liU/comp-8005-assign-4/internal/shared"
)

func (w *Worker) routeHeartbeat(m shared.Message, s string) (shared.Message, error) {
	w.Logger.Info(m.Message)

	total := w.getTotalAttempts()
	delta := total - w.lastAttemptsSent

	w.lastAttemptsSent = total

	return shared.Message{
			Type:      shared.MessageHeartbeat,
			Payload:   shared.PayloadHearbeat{TotalTested: total, DeltaTested: delta, ActiveThreads: w.Config.Threads},
			Timestamp: time.Now(),
		},
		nil
}
