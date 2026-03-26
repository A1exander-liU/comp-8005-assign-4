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
	resp chan int
}

type passwordResponse struct {
	password string
}

// HandleJobV1 performs password cracking utilizing a shared password index. Each thread will
// request an index to the next thread. This ensures passwords are attempted sequentially, simplifying
// storage of checkpoint progress.
func (w *Worker) HandleJobV1(payload shared.PayloadJobDetails, dispatchTime time.Duration) {
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
			if ((passwordIndex-payload.ChunkStart)+1)%uint64(payload.CheckpointAttempts) == 0 {
				checkpoint += 1
				fmt.Printf("[leader] checkpoint %d\n", checkpoint)
			}

			select {
			case passwordRequest := <-passwordRequests:
				passwordRequest.resp <- int(passwordIndex)
			case result := <-done:
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
		wg.Go(func() {
			for {
				passwordRequest := passwordRequest{resp: make(chan int)}
				passwordRequests <- passwordRequest

				passwordID, ok := <-passwordRequest.resp

				// closed channel
				if !ok {
					fmt.Printf("[worker %d] channel closed\n", i)
					return
				}

				password := shared.EncodeBase(uint64(passwordID), shared.SearchSpace)
				if digest.Match(password) {
					fmt.Printf("[worker %d] found password: %s\n", i, password)
					done <- passwordResponse{password: password}
					return
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

// HandleJobV2 performs password cracking by creating static partitions of the passowrds
// and giving each to a thread.
func (w *Worker) HandleJobV2(payload shared.PayloadJobDetails, dispatchTime time.Duration) {
}
