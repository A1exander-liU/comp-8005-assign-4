package main

import (
	"flag"
	"fmt"
	"net"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"go.uber.org/zap"
)

type Worker struct{}

type controller struct {
	server net.Listener
	logger *zap.Logger

	workers map[string]net.Conn

	quit chan any
	wg   sync.WaitGroup
}

type settings struct {
	shadowfile string
	username   string
	port       int
}

func newController() *controller {
	c := controller{
		workers: make(map[string]net.Conn),
		quit:    make(chan any),
	}

	c.wg.Add(1)
	return &c
}

func (c *controller) cleanup() {
	c.logger.Info("Doing graceful server shutdown")

	if c.server == nil {
		return
	}

	close(c.quit)
	_ = c.server.Close()
	c.wg.Wait()
}

func setupServer(logger *zap.Logger, port int) net.Listener {
	address := fmt.Sprintf("[::]:%d", port)

	server, err := net.Listen("tcp", address)
	if err != nil {
		logger.Fatal("Server failed to start", zap.Error(err))
	}

	logger.Info("Server started listening", zap.String("address", server.Addr().String()))

	return server
}

func handleArguments(settings *settings) {
	if settings.shadowfile == "" {
		fmt.Println("Error: -f is required")
		flag.Usage()
		os.Exit(1)
	}
	if settings.username == "" {
		fmt.Println("Error: -u is required")
		flag.Usage()
		os.Exit(1)
	}
	if settings.port < 1 || settings.port > 65535 {
		fmt.Println("Error: -p is required and in the range 1 - 65535 (inclusive)")
		flag.Usage()
		os.Exit(1)
	}
}

func parseArguments() settings {
	var settings settings

	flag.StringVar(&settings.shadowfile, "f", "", "path to shadowfile")
	flag.StringVar(&settings.username, "u", "", "username to be cracked")
	flag.IntVar(&settings.port, "p", 0, "port number to listen on")

	flag.Parse()

	return settings
}

func main() {
	logger, _ := zap.NewDevelopment()
	defer logger.Sync().Error()
	controller := newController()
	controller.logger = logger

	settings := parseArguments()
	handleArguments(&settings)

	controller.logger.Info("Settings",
		zap.String("shadowfile", settings.shadowfile),
		zap.String("username", settings.username),
		zap.Int("port", settings.port),
	)

	server := setupServer(controller.logger, settings.port)
	controller.server = server

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-stop

		controller.cleanup()

		os.Exit(0)
	}()

	for {
		conn, err := controller.server.Accept()
		if err != nil {
			select {
			case <-controller.quit:
				return
			default:
				controller.logger.Error("Accept connection failed", zap.Error(err))
			}

			continue
		}

		controller.wg.Add(1)
		controller.logger.Info("Connection received", zap.String("address", conn.LocalAddr().String()))
		controller.wg.Done()
	}
}
