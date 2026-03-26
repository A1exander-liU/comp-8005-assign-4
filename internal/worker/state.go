package worker

import (
	"maps"

	"github.com/A1exander-liU/comp-8005-assign-2/internal/shared"
)

const StateFileLocation string = "data/state.json"

type WorkerState struct {
	ID            string `json:"id"`
	ChunkID       int    `json:"chunk_id"`
	PasswordIndex int    `json:"password_index"`

	CompeletedPasswords map[uint64]bool `json:"completed_passwords"`

	Payload shared.PayloadJobDetails `json:"payload"`
}

func (w *Worker) getTotalAttempts() int {
	// send heartbeat, update the lastAttemptsSent
	// during crash lastAttemptsSent will be reset to 0
	// during reconnect set it current total
	var attempts int

	w.mu.Lock()
	defer w.mu.Unlock()

	attempts = len(w.state.CompeletedPasswords)
	return attempts
}

func (w *Worker) addAttempt(passwordIndex uint64) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.state.CompeletedPasswords[passwordIndex] = true
}

func (w *Worker) getAttempt(passwordIndex uint64) (bool, bool) {
	w.mu.Lock()
	defer w.mu.Unlock()
	result, ok := w.state.CompeletedPasswords[passwordIndex]
	return result, ok
}

func (w *Worker) getAttemptsCopy() map[uint64]bool {
	w.mu.Lock()
	defer w.mu.Unlock()

	completedPasswordsCopy := map[uint64]bool{}
	maps.Copy(completedPasswordsCopy, w.state.CompeletedPasswords)

	return completedPasswordsCopy
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
		ID:                  "",
		ChunkID:             0,
		PasswordIndex:       0,
		CompeletedPasswords: map[uint64]bool{},
		Payload:             shared.PayloadJobDetails{},
	}
}

func InitialStateWithID(id string) WorkerState {
	return WorkerState{
		ID:                  id,
		ChunkID:             0,
		PasswordIndex:       0,
		CompeletedPasswords: map[uint64]bool{},
		Payload:             shared.PayloadJobDetails{},
	}
}
