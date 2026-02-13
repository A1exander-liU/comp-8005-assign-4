package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/A1exander-liU/comp-8005-assign-2/internal/controller"
	"github.com/A1exander-liU/comp-8005-assign-2/internal/shared"
	"go.uber.org/zap"
)

func handleArguments(config *controller.Config) {
	if config.Shadowfile == "" {
		fmt.Println("Error: -f is required")
		flag.Usage()
		os.Exit(1)
	}
	if config.Username == "" {
		fmt.Println("Error: -u is required")
		flag.Usage()
		os.Exit(1)
	}
	if config.Port < 1 || config.Port > 65535 {
		fmt.Println("Error: -p is required and must be in range: 1 - 65535 (inclusive)")
		flag.Usage()
		os.Exit(1)
	}

	if config.HeartbeatSeconds < 1 {
		fmt.Println("Error: -b must be a non-zero positive integer")
		flag.Usage()
		os.Exit(1)
	}
}

func parseShadowfile(shadowfile, username string) shared.ShadowData {
	contents, err := os.ReadFile(shadowfile)
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

		if user != username {
			continue
		}

		shadow, err := shared.ParseHash(hash)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		shadow.Username = user
		return shadow
	}

	fmt.Println("Failed to find user:", username)
	os.Exit(1)

	return shared.ShadowData{}
}

func parseArguments() controller.Config {
	var config controller.Config

	flag.StringVar(&config.Shadowfile, "f", "", "path to shadowfile")
	flag.StringVar(&config.Username, "u", "", "username to be cracked")
	flag.IntVar(&config.Port, "p", 0, "port number to listen on")
	flag.IntVar(&config.HeartbeatSeconds, "b", 0, "period (seconds) to send a heartbeat")

	flag.Parse()

	return config
}

func main() {
	shared.RegisterMessages()

	logPath := filepath.Join("logs", "log")
	if _, err := os.Stat(logPath); !os.IsNotExist(err) {
		if err = os.Remove(logPath); err != nil {
			fmt.Println("Failed to remove file:", err)
			os.Exit(1)
		}
	}

	cfg := zap.NewDevelopmentConfig()
	cfg.OutputPaths = []string{"stdout", "./logs/log"}
	cfg.ErrorOutputPaths = []string{"stderr", "./logs/log"}

	logger := zap.Must(cfg.Build())

	config := parseArguments()
	handleArguments(&config)

	parseStart := time.Now()
	shadowData := parseShadowfile(config.Shadowfile, config.Username)
	parseDuration := time.Since(parseStart)

	controller := controller.NewController(logger, shadowData, config.HeartbeatSeconds)
	controller.LatencyParse = parseDuration

	controller.SetupServer(config.Port)

	for {
		conn, err := controller.AcceptConnection()
		if err != nil {
			controller.Logger.Info("Failed to accept connection", zap.Error(err))
		}

		go controller.HandleConnection(conn)
	}
}
