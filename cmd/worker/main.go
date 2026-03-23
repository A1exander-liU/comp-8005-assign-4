package main

import (
	"github.com/A1exander-liU/comp-8005-assign-2/internal/shared"
	"github.com/A1exander-liU/comp-8005-assign-2/internal/worker"
	"go.uber.org/zap"
)

func main() {
	shared.RegisterMessages()

	cfg := zap.NewDevelopmentConfig()
	cfg.DisableCaller = true

	logger := zap.Must(cfg.Build())

	worker := worker.NewWorker(logger)
	config := worker.ParseArguments()
	worker.HandleArguments(&config)

	worker.SetupServer()
	worker.HandleConnection()
}
