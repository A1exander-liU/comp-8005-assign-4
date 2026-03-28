package worker

import (
	"github.com/A1exander-liU/comp-8005-assign-2/internal/shared"
)

type WorkerState struct {
	ID            string `json:"id"`
	ChunkID       int    `json:"chunk_id"`
	PasswordIndex int    `json:"password_index"`

	Payload shared.PayloadJobDetails `json:"payload"`
}

func (w *Worker) getTotalAttempts() int {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.state.PasswordIndex - int(w.state.Payload.ChunkStart)
}

func LoadState(path string) (WorkerState, error) {
	var state WorkerState

	if err := shared.ReadFromJSON(path, &state); err != nil {
		return WorkerState{}, err
	}

	return state, nil
}

func SaveState(path string, state WorkerState) error {
	return shared.WriteToJSON(path, state)
}

func InitialState() WorkerState {
	return WorkerState{
		ID:            "",
		ChunkID:       0,
		PasswordIndex: 0,
		Payload:       shared.PayloadJobDetails{},
	}
}

func InitialStateWithID(id string) WorkerState {
	return WorkerState{
		ID:            id,
		ChunkID:       0,
		PasswordIndex: 0,
		Payload:       shared.PayloadJobDetails{},
	}
}
