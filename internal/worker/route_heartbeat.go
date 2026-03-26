package worker

import "github.com/A1exander-liU/comp-8005-assign-2/internal/shared"

func (w *Worker) routeHeartbeat(m shared.Message, s string) (shared.Message, error) {
	return shared.Message{
			Type:    shared.MessageHeartbeat,
			Payload: shared.PayloadHearbeat{TotalTested: 0, DeltaTested: 0, ActiveThreads: w.Config.Threads},
		},
		nil
}
