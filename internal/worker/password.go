package worker

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/A1exander-liU/comp-8005-assign-2/internal/shared"
	"github.com/go-crypt/crypt"
	"go.uber.org/zap"
)

func (w *Worker) buildHash(payload shared.PayloadJobDetails) string {
	sections := make([]string, 0)
	if payload.Algorithm != "" {
		sections = append(sections, fmt.Sprintf("$%s", payload.Algorithm))
	}
	if payload.Parameters != "" {
		sections = append(sections, payload.Parameters)
	}
	if payload.Salt != "" {
		sections = append(sections, payload.Salt)
	}
	if payload.Hash != "" {
		sections = append(sections, payload.Hash)
	}

	fullHash := strings.Join(sections, "$")
	return fullHash
}

// HandleJobV1 performs password cracking utilizing a shared password index. Each thread will
// request an index to the next thread. This ensures passwords are attempted sequentially, simplifying
// storage of checkpoint progress.
func (w *Worker) handleJobV1(payload shared.PayloadJobDetails, dispatchStart, dispatchEnd time.Time) {
	w.Logger.Info("Cracking started...")
	fullHash := w.buildHash(payload)

	decoder, _ := crypt.NewDecoderAll()
	digest, err := decoder.Decode(fullHash)
	if err != nil {
		return
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var (
		index       atomic.Uint64 // next password index to try
		found       atomic.Bool   // true once a match is seen
		foundPasswd atomic.Value  // stores the string once found
		wg          sync.WaitGroup

		localAttempts atomic.Int64
	)

	index.Store(uint64(w.state.PasswordIndex))
	passwordCrackStart := time.Now()

	for i := range w.Config.Threads {
		wg.Add(1)
		workerID := i + 1

		go func() {
			defer wg.Done()

			for {
				// Check cancellation first
				select {
				case <-ctx.Done():
					// fmt.Printf("[worker %d] cancelled\n", workerID)
					return
				default:
				}

				// Atomically claim the next password while incrementing for other threads
				idx := index.Add(1) - 1
				w.mu.Lock()
				w.state.PasswordIndex = int(idx + 1)
				w.mu.Unlock()

				if idx >= payload.ChunkEnd {
					// fmt.Printf("[worker %d] no more passwords\n", workerID)
					return
				}

				password := shared.EncodeBase(idx, shared.SearchSpace)
				if digest.Match(password) {
					w.Logger.Info(fmt.Sprintf("[worker %d] found password: %s\n", workerID, password))
					foundPasswd.Store(password)
					found.Store(true)
					cancel() // signal all other workers
					return
				}

				n := localAttempts.Add(1)
				if n%int64(payload.CheckpointAttempts) == 0 {
					w.router.Send(shared.Message{
						Version:   shared.MessageVersion,
						Type:      shared.MessageJobCheckpoint,
						Timestamp: time.Now(),
						Message:   "Send checkpoint",
						Payload: shared.PayloadCheckpoint{
							ChunkID:    payload.ChunkID,
							ChunkIndex: idx,
						},
					})
				}
			}
		}()
	}

	wg.Wait()

	passwordCrackDuration := time.Since(passwordCrackStart)

	var foundPassword string
	if v := foundPasswd.Load(); v != nil {
		foundPassword = v.(string)
	}

	w.Logger.Info(fmt.Sprintf("Done in %s — password: %q\n", passwordCrackDuration, foundPassword))
	w.Logger.Info("Job results submitted", zap.Int("chunkID", payload.ChunkID))

	var dispatchTime time.Duration
	if dispatchStart.Equal(time.Time{}) {
		dispatchTime = 0 * time.Second
	} else {
		dispatchTime = dispatchEnd.Sub(dispatchStart)
	}

	if foundPassword == "" {
		w.router.Send(
			shared.Message{
				Version:   shared.MessageVersion,
				Type:      shared.MessageJobResults,
				Timestamp: time.Now(),
				Message:   "Job results sent",
				Payload: shared.PayloadJobResults{
					Password:  foundPassword,
					CrackTime: passwordCrackDuration, DispatchTime: dispatchTime,
					Err:     "password not found in chunk",
					ChunkID: payload.ChunkID,
				},
			},
		)
	} else {
		w.router.Send(
			shared.Message{
				Version:   shared.MessageVersion,
				Type:      shared.MessageJobResults,
				Timestamp: time.Now(),
				Message:   "Job results sent",
				Payload: shared.PayloadJobResults{
					Password:  foundPassword,
					CrackTime: passwordCrackDuration, DispatchTime: dispatchTime,
					ChunkID: payload.ChunkID,
				},
			},
		)
	}
}
