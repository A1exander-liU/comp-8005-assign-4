package worker

import (
	"fmt"
	"strings"
	"sync"
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

type passwordRequest struct {
	resp chan uint64
}

type passwordResponse struct {
	password      string
	passwordIndex uint64
}

// HandleJobV1 performs password cracking utilizing a shared password index. Each thread will
// request an index to the next thread. This ensures passwords are attempted sequentially, simplifying
// storage of checkpoint progress.
func (w *Worker) handleJobV1(payload shared.PayloadJobDetails) {
	fmt.Println("Cracking started...")
	fullHash := w.buildHash(payload)

	// make sure password hash is correct format
	decoder, _ := crypt.NewDecoderAll()
	digest, err := decoder.Decode(fullHash)
	if err != nil {
		return
	}

	var wg sync.WaitGroup

	passwordRequests := make(chan passwordRequest, w.Config.Threads)
	done := make(chan passwordResponse, 1)
	result := make(chan string)

	// coordinator thread
	go func() {
		var password string
		checkpoint := 0

	loop:
		for passwordIndex := payload.ChunkStart; passwordIndex < payload.ChunkEnd; passwordIndex++ {
			// skip already completed passwords
			if _, ok := w.state.CompeletedPasswords[int(passwordIndex)]; ok {
				continue
			}

			n := len(w.state.CompeletedPasswords)
			if n > 0 && n%payload.CheckpointAttempts == 0 {
				checkpoint += 1
				fmt.Printf("[leader] checkpoint %d\n", checkpoint)
			}

			select {
			case passwordRequest := <-passwordRequests:
				passwordRequest.resp <- passwordIndex
			case result := <-done:
				w.state.CompeletedPasswords[int(result.passwordIndex)] = true

				if result.password == "" {
					continue
				}

				password = result.password
				fmt.Printf("[leader] password found: %s\n", result.password)
				break loop
			}
		}

		// drain requests and close the channels
		for request := range passwordRequests {
			close(request.resp)
		}
		result <- password
	}()

	passwordCrackStart := time.Now()

	// worker threads
	for i := range w.Config.Threads {
		workerThreadID := i + 1
		wg.Go(func() {
			for {
				passwordRequest := passwordRequest{resp: make(chan uint64)}
				passwordRequests <- passwordRequest

				passwordID, ok := <-passwordRequest.resp

				// closed channel
				if !ok {
					fmt.Printf("[worker %d] channel closed\n", workerThreadID)
					return
				}

				password := shared.EncodeBase(passwordID, shared.SearchSpace)
				if digest.Match(password) {
					fmt.Printf("[worker %d] found password: %s\n", workerThreadID, password)
					done <- passwordResponse{password: password, passwordIndex: passwordID}
					return
				} else {
					done <- passwordResponse{password: "", passwordIndex: passwordID}
				}
			}
		})
	}

	wg.Wait()
	close(passwordRequests)
	foundPassword := <-result

	passwordCrackDuration := time.Since(passwordCrackStart)

	if foundPassword == "" {
		fmt.Printf("done in %v: password not found\n", passwordCrackDuration)
	} else {
		fmt.Printf("done in %v: %s\n", passwordCrackDuration, foundPassword)
	}
}
