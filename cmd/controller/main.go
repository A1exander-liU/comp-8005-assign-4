package main

import (
	"errors"
	"net"

	"github.com/A1exander-liU/comp-8005-assign-2/internal/controller"
	"github.com/A1exander-liU/comp-8005-assign-2/internal/shared"
	"go.uber.org/zap"
)

func main() {
	shared.RegisterMessages()

	// logPath := filepath.Join("logs", "log")
	// if _, err := os.Stat(logPath); !os.IsNotExist(err) {
	// 	if err = os.Remove(logPath); err != nil {
	// 		fmt.Println("Failed to remove file:", err)
	// 		os.Exit(1)
	// 	}
	// }

	cfg := zap.NewDevelopmentConfig()
	// cfg.OutputPaths = []string{"stdout", "./logs/log"}
	// cfg.ErrorOutputPaths = []string{"stderr", "./logs/log"}
	logger := zap.Must(cfg.Build())

	c := controller.NewController(logger)
	config := c.ParseArguments()
	c.HandleArguments(config)
	c.SetupServer()

	for {
		conn, err := c.AcceptConnection()
		if errors.Is(err, net.ErrClosed) {
			return
		}
		if err != nil {
			c.Logger.Info("Failed to accept connection", zap.Error(err))
			continue
		}

		go c.HandleConnection(conn)
	}
}
