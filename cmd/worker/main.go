package main

import (
	"github.com/A1exander-liU/comp-8005-assign-4/internal/shared"
	"github.com/A1exander-liU/comp-8005-assign-4/internal/worker"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func main() {
	shared.RegisterMessages()

	cfg := zap.NewDevelopmentConfig()
	cfg.DisableCaller = true
	cfg.DisableStacktrace = true
	cfg.Level.SetLevel(zapcore.InfoLevel)

	logger := zap.Must(cfg.Build())

	worker := worker.NewWorker(logger)
	config := worker.ParseArguments()
	worker.HandleArguments(&config)

	worker.SetupServer()
	worker.Start()
	// worker.HandleConnection()
}
