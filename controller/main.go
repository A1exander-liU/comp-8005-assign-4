package main

import (
	"encoding/gob"
	"flag"
	"fmt"
	"net"
	"os"
	"os/signal"
	"sync"
	"syscall"

	utils "github.com/A1exander-liU/comp-8005-assign-1"
	"go.uber.org/zap"
)

type controller struct {
	server net.Listener

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
		wg:      sync.WaitGroup{},
	}

	c.wg.Add(1)
	return &c
}

func (c *controller) cleanup(logger *zap.Logger) {
	logger.Info("Doing graceful server shutdown")

	if c.server == nil {
		return
	}

	close(c.quit)
	_ = c.server.Close()
	c.wg.Wait()
}

func displayJobResults(logger *zap.Logger, m utils.Message) {
	logger.Info("Displaying crack results",
		zap.String("version", m.Version),
		zap.String("type", m.Type),
		zap.String("message", m.Message),
		zap.String("result", m.Result),
	)
}
func handleJobResults(logger *zap.Logger, m utils.Message) {}
func sendJob(logger *zap.Logger, encoder *gob.Encoder, s utils.ShadowData) {
	m := utils.Message{
		Version: "1", Type: "job.details", Message: "Cracking details",
		Data: s,
	}
	_ = encoder.Encode(m)
	logger.Info("Sending job details to worker",
		zap.String("version", m.Version),
		zap.String("type", m.Type),
		zap.String("message", m.Message),
	)
}

func sendRegistrationConfirmation(logger *zap.Logger, encoder *gob.Encoder) {
	m := utils.Message{
		Version: "1", Type: "registration.confirm", Message: "Sending registration confirmation",
	}
	_ = encoder.Encode(m)

	logger.Info("Sending registration confirmation to worker",
		zap.String("version", m.Version),
		zap.String("type", m.Type),
		zap.String("message", m.Message),
	)
}

func handleRegistration(logger *zap.Logger) {
	logger.Info("New worker connected")
}

func handleConnection(logger *zap.Logger, conn net.Conn, s utils.ShadowData) {
	if conn == nil {
		return
	}
	decoder := gob.NewDecoder(conn)
	encoder := gob.NewEncoder(conn)

	for {
		var m utils.Message

		if err := decoder.Decode(&m); err != nil {
			logger.Error("Failed to decode", zap.Error(err))
			return
		}

		logger.Info("Message received",
			zap.String("version", m.Version),
			zap.String("type", m.Type),
			zap.String("message", m.Message),
		)

		switch m.Type {
		case "connection.terminate":
			_ = conn.Close()
			return
		case "registration.request":
			handleRegistration(logger)
			sendRegistrationConfirmation(logger, encoder)
		case "registration.confirm":
			sendJob(logger, encoder, s)
		case "job.results":
			handleJobResults(logger, m)
			displayJobResults(logger, m)
		}
	}
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

func parseShadowFile(shadowfile, username string) utils.ShadowData {
	return utils.ShadowData{
		Algorithm: "y", Hash: shadowfile, Salt: username,
	}
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

	settings := parseArguments()
	handleArguments(&settings)

	logger.Info("Settings",
		zap.String("shadowfile", settings.shadowfile),
		zap.String("username", settings.username),
		zap.Int("port", settings.port),
	)

	shadowData := parseShadowFile(settings.shadowfile, settings.username)

	server := setupServer(logger, settings.port)
	controller.server = server

	// handle ctrl+c terminations
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-stop
		controller.cleanup(logger)
		os.Exit(0)
	}()

	for {
		conn, err := controller.server.Accept()
		if err != nil {
			select {
			case <-controller.quit:
				return
			default:
				logger.Error("Accept connection failed", zap.Error(err))
			}

			continue
		}

		controller.wg.Add(1)
		controller.wg.Go(func() {
			logger.Info("Connection received", zap.String("address", conn.RemoteAddr().String()))
			handleConnection(logger, conn, shadowData)
			controller.wg.Done()
		})
	}
}
