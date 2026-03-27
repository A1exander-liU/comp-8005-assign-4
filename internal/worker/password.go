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
func (w *Worker) handleJobV1(payload shared.PayloadJobDetails) {
	fmt.Println("Cracking started...")
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

		checkpoint    int
		checkpointMu  sync.Mutex
		localAttempts atomic.Int64
	)

	index.Store(payload.ChunkStart)

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
					fmt.Printf("[worker %d] cancelled\n", workerID)
					return
				default:
				}

				// Atomically claim the next index
				idx := index.Add(1) - 1

				// Skip already-attempted indices
				for {
					if _, attempted := w.getAttempt(idx); !attempted {
						break
					}
					idx = index.Add(1) - 1
				}

				if idx >= payload.ChunkEnd {
					fmt.Printf("[worker %d] no more passwords\n", workerID)
					return
				}

				password := shared.EncodeBase(idx, shared.SearchSpace)

				if digest.Match(password) {
					fmt.Printf("[worker %d] found password: %s\n", workerID, password)
					foundPasswd.Store(password)
					found.Store(true)
					cancel() // signal all other workers
					return
				}

				w.addAttempt(idx)

				// Checkpoint handling
				n := localAttempts.Add(1)
				if n%int64(payload.CheckpointAttempts) == 0 {
					checkpointMu.Lock()
					checkpoint++
					cp := checkpoint
					checkpointMu.Unlock()

					fmt.Printf("[checkpoint %d]\n", cp)
					w.router.Send(shared.Message{
						Version:   shared.MessageVersion,
						Type:      shared.MessageJobCheckpoint,
						Timestamp: time.Now(),
						Message:   "Send checkpoint",
						Payload: shared.PayloadCheckpoint{
							ChunkID:            payload.ChunkID,
							CompletedPasswords: w.getAttemptsCopy(),
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

	fmt.Printf("Done in %s — password: %q\n", passwordCrackDuration, foundPassword)
	if foundPassword == "" {
		w.router.Send(
			shared.Message{
				Version:   shared.MessageVersion,
				Type:      shared.MessageJobResults,
				Timestamp: time.Now(),
				Message:   "Job results sent",
				Payload: shared.PayloadJobResults{
					Password:  foundPassword,
					CrackTime: passwordCrackDuration, DispatchTime: 0 * time.Second,
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
					CrackTime: passwordCrackDuration, DispatchTime: 0 * time.Second,
					ChunkID: payload.ChunkID,
				},
			},
		)
	}
}
