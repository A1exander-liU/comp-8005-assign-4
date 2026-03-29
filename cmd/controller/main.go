package main

import (
	"errors"
	"net"

	"github.com/A1exander-liU/comp-8005-assign-4/internal/controller"
	"github.com/A1exander-liU/comp-8005-assign-4/internal/shared"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func main() {
	shared.RegisterMessages()

	cfg := zap.NewDevelopmentConfig()
	cfg.DisableCaller = true
	cfg.Level.SetLevel(zapcore.InfoLevel)

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
