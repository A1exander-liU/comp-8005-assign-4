package worker

import (
	"flag"
	"fmt"
	"os"

	"github.com/A1exander-liU/comp-8005-assign-2/internal/shared"
)

func (w *Worker) ParseArguments() Config {
	var config Config

	fs := flag.NewFlagSet("worker CLI", flag.ExitOnError)

	fs.StringVar(&config.ControllerIP, "c", "", "controller ip")
	fs.IntVar(&config.ControllerPort, "p", 0, "controller port number")
	fs.IntVar(&config.Threads, "t", 1, "thread count for password cracking")
	fs.StringVar(&config.CheckpointFile, "f", "./data/state.json", "path to checkpoint file")

	w.fs = fs

	if err := w.fs.Parse(os.Args[1:]); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	return config
}

func (w *Worker) HandleArguments(config *Config) {
	if config.ControllerIP == "" {
		fmt.Println("Error: -c is required")
		w.fs.Usage()
		os.Exit(1)
	}
	if config.ControllerPort < 1 || config.ControllerPort > 65535 {
		fmt.Println("Error: -p is required and in the range 1 - 65535 (inclusive)")
		flag.Usage()
		os.Exit(1)
	}

	if config.Threads < 1 {
		fmt.Println("Error: -t must be a non-zero positive integer")
		w.fs.Usage()
		os.Exit(1)
	}

	result := shared.ParseAddress(config.ControllerIP, config.ControllerPort)
	if result == "" {
		fmt.Println("controller ip is not in correct format")
		w.fs.Usage()
		os.Exit(1)
	}

	w.Config = *config
}
