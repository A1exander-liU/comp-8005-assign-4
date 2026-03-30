package controller

import (
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/A1exander-liU/comp-8005-assign-4/internal/shared"
)

// ParseArguments parses command line arguments.
func (c *Controller) ParseArguments() Config {
	var config Config
	fs := flag.NewFlagSet("controller CLI", flag.ExitOnError)

	fs.StringVar(&config.Shadowfile, "f", "", "path to shadowfile")
	fs.StringVar(&config.Username, "u", "", "username whose password to be cracked")
	fs.IntVar(&config.Port, "p", 0, "port number to listen on")
	fs.IntVar(&config.HeartbeatSeconds, "b", 0, "period (in seconds) to send a heartbeat")
	fs.IntVar(&config.ChunkSize, "c", 0, "chunk size of each cracking task for a worker")
	fs.IntVar(&config.CheckpointAttempts, "k", 0, "number of attempts before worker sends a checkpoint")
	c.fs = fs

	if err := fs.Parse(os.Args[1:]); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	return config
}

// HandleArguments performs validation on the arguments, the program will exit
// and print out a usage if any of the arguments failed validation.
func (c *Controller) HandleArguments(config Config) {
	c.metric.SetMetric(MetricParseStart, time.Now())
	if config.Shadowfile == "" {
		fmt.Println("Error: -f is required")
		c.fs.Usage()
		os.Exit(1)
	}
	if config.Username == "" {
		fmt.Println("Error: -u is required")
		c.fs.Usage()
		os.Exit(1)
	}
	if config.Port < 1 || config.Port > 65535 {
		fmt.Println("Error: -p is required and must be in range: 1 - 65535 (inclusive)")
		c.fs.Usage()
		os.Exit(1)
	}

	if config.HeartbeatSeconds < 1 {
		fmt.Println("Error: -b must be a non-zero positive integer")
		c.fs.Usage()
		os.Exit(1)
	}

	if config.ChunkSize < 1 {
		fmt.Println("Error: -c must be a non-zero positive integer")
		c.fs.Usage()
		os.Exit(1)
	}

	if config.CheckpointAttempts < 1 {
		fmt.Println("Error: -k must be a non-zero positive integer")
		c.fs.Usage()
		os.Exit(1)
	}

	c.Config = config
	c.parseShadowFile()
	c.metric.SetMetric(MetricParseEnd, time.Now())
}

// parseShadowFile reads to the shadowfile to extracted the password hash elements
// of the desired user.
//
// This will return with an error if:
//   - it failed to read the shadowfile
//   - it could not find the user
func (c *Controller) parseShadowFile() {
	foundUser := false

	contents, err := os.ReadFile(c.Config.Shadowfile)
	if err != nil {
		fmt.Println("Failed to read shadowfile:", err)
		os.Exit(1)
	}

	entries := strings.SplitSeq(string(contents), "\n")
	for entry := range entries {
		user, hash, found := strings.Cut(entry, ":")
		if !found {
			continue
		}

		if user != c.Config.Username {
			continue
		}

		foundUser = true
		shadow, err := shared.ParseHash(hash)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		shadow.Username = user
		c.ShadowData = shadow
	}

	if !foundUser {
		fmt.Println("Failed to find user:", c.Config.Username)
		os.Exit(1)
	}
}
