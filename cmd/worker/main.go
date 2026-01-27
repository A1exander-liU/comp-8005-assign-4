package main

import (
	"fmt"
	"os"

	"github.com/A1exander-liU/comp-8005-assign-1/internal/worker"
	"go.uber.org/zap"
)

func main() {
	logger, err := zap.NewDevelopment()
	if err != nil {
		fmt.Println("Failed to create logger:", err)
		os.Exit(1)
	}

	worker := worker.NewWorker(logger)
	worker.HandleConnection()
}
