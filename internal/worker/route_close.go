package worker

import "github.com/A1exander-liU/comp-8005-assign-4/internal/shared"

func (w *Worker) routeClose(m shared.Message, id string) (shared.Message, error) {
	w.cleanup()
	return shared.Message{}, nil
}
