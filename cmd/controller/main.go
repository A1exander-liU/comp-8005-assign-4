package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/A1exander-liU/comp-8005-assign-1/internal/controller"
	"github.com/A1exander-liU/comp-8005-assign-1/internal/shared"
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

	flag.Parse()

	return config
}

func main() {
	shared.RegisterMessages()

	logger, err := zap.NewDevelopment()
	if err != nil {
		fmt.Println("Failed to create logger:", err)
		os.Exit(1)
	}
	controller := controller.NewController(logger, shared.ShadowData{})

	controller.Logger.Info("Started parsing")
	config := parseArguments()
	handleArguments(&config)

	shadowData := parseShadowfile(config.Shadowfile, config.Username)
	logger.Info("Finished parsing")
	controller.ShadowData = shadowData

	controller.SetupServer(config.Port)

	for {
		conn, err := controller.AcceptConnection()
		if err != nil {
			controller.Logger.Info("Failed to accept connection", zap.Error(err))
		}

		go controller.HandleConnection(conn)
	}
}
