package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/A1exander-liU/comp-8005-assign-2/internal/shared"
	"github.com/A1exander-liU/comp-8005-assign-2/internal/worker"
	"go.uber.org/zap"
)

func handleArguments(config *worker.Config) {
	if config.ControllerIP == "" {
		fmt.Println("Error: -c is required")
		flag.Usage()
		os.Exit(1)
	}
	if config.ControllerPort < 1 || config.ControllerPort > 65535 {
		fmt.Println("Error: -p is required and in the range 1 - 65535 (inclusive)")
		flag.Usage()
		os.Exit(1)
	}

	if config.Threads < 1 {
		fmt.Println("Error: -t must be a non-zero positive integer")
		flag.Usage()
		os.Exit(1)
	}

	result := shared.ParseAddress(config.ControllerIP, config.ControllerPort)
	if result == "" {
		fmt.Println("controller ip is not in correct format")
		flag.Usage()
		os.Exit(1)
	}
}

func parseArguments() worker.Config {
	var config worker.Config

	flag.StringVar(&config.ControllerIP, "c", "", "controller ip")
	flag.IntVar(&config.ControllerPort, "p", 0, "controller port number")
	flag.IntVar(&config.Threads, "t", 1, "thread count for password cracking")

	flag.Parse()

	return config
}

func main() {
	shared.RegisterMessages()

	cfg := zap.NewDevelopmentConfig()

	logger := zap.Must(cfg.Build())

	config := parseArguments()
	handleArguments(&config)

	worker := worker.NewWorker(logger)
	worker.SetupServer(config)
	worker.HandleConnection()
}
