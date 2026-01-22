package main

import (
	"encoding/gob"
	"flag"
	"fmt"
	"log"
	"net"
	"os"

	utils "github.com/A1exander-liU/comp-8005-assign-1"
	"go.uber.org/zap"
)

type settings struct {
	controllerIP      string
	controllerPort    int
	controllerAddress string
}

func crackPassword(message utils.Message) string {
	log.Println(message)
	log.Println("cracking password")

	return "cracked"
}

func sendTermination(logger *zap.Logger, encoder *gob.Encoder) {
	m := utils.Message{Version: "1", Type: "DONE", Message: "Finished"}
	_ = encoder.Encode(m)
	logger.Info("Send connection termination",
		zap.String("version", m.Version),
		zap.String("type", m.Type),
		zap.String("message", m.Message),
	)
}

func sendJobResults(logger *zap.Logger, encoder *gob.Encoder, result string) {
	m := utils.Message{Version: "1", Type: "JOB.RESULTS", Message: result}
	_ = encoder.Encode(m)
	logger.Info("Send job results",
		zap.String("version", m.Version),
		zap.String("type", m.Type),
		zap.String("message", m.Message),
	)
}

func handleJob(logger *zap.Logger, decoder *gob.Decoder) utils.Message {
	for {
		var m utils.Message

		if err := decoder.Decode(&m); err != nil {
			logger.Error("Failed to decode", zap.Error(err))
			continue
		}

		logger.Info("Message received",
			zap.String("version", m.Version),
			zap.String("type", m.Type),
			zap.String("message", m.Message),
		)

		if m.Type == "JOB.DETAILS" {
			return m
		}
	}
}

func handleRegistrationConfirmation(logger *zap.Logger, decoder *gob.Decoder) {
	for {
		var m utils.Message

		if err := decoder.Decode(&m); err != nil {
			logger.Error("Failed to decode", zap.Error(err))
			continue
		}

		logger.Info("Message received",
			zap.String("version", m.Version),
			zap.String("type", m.Type),
			zap.String("message", m.Message),
		)

		if m.Type == "REGISTRATION.CONFIRMATION" {
			return
		}
	}
}

func sendRegistration(logger *zap.Logger, encoder *gob.Encoder) {
	m := utils.Message{Version: "1", Type: "REGISTRATION", Message: "Requesting registration"}
	_ = encoder.Encode(m)
	logger.Info("Sending resgistration",
		zap.String("version", m.Version),
		zap.String("type", m.Type),
		zap.String("message", m.Message),
	)
}

func setupServer(controllerIP string, controllerPort int) net.Conn {
	utils.ParseAddress(controllerIP, controllerPort)
	controllerAddress := fmt.Sprintf("%s:%d", controllerIP, controllerPort)
	fmt.Println(controllerAddress)

	conn, err := net.Dial("tcp", controllerAddress)
	if err != nil {
		log.Fatal(err)
	}

	return conn
}

func handleArguments(settings *settings) {
	if settings.controllerIP == "" {
		fmt.Println("Error: -c is required")
		flag.Usage()
		os.Exit(1)
	}
	if settings.controllerPort < 1 || settings.controllerPort > 65535 {
		fmt.Println("Error: -p is required and in the range 1 - 65535 (inclusive)")
		flag.Usage()
		os.Exit(1)
	}

	result := utils.ParseAddress(settings.controllerIP, settings.controllerPort)
	if result == "" {
		fmt.Println("controller ip is not in correct format")
		flag.Usage()
		os.Exit(1)
	}
}

func parseArguments() settings {
	var settings settings

	flag.StringVar(&settings.controllerIP, "c", "", "controller ip")
	flag.IntVar(&settings.controllerPort, "p", 0, "controller port number")

	flag.Parse()

	return settings
}

func main() {
	logger, _ := zap.NewDevelopment()
	defer logger.Sync().Error()

	settings := parseArguments()
	handleArguments(&settings)

	logger.Info("Settings",
		zap.String("controller ip", settings.controllerIP),
		zap.Int("controller port", settings.controllerPort),
	)

	conn := setupServer(settings.controllerIP, settings.controllerPort)

	encoder := gob.NewEncoder(conn)
	decoder := gob.NewDecoder(conn)

	sendRegistration(logger, encoder)
	handleRegistrationConfirmation(logger, decoder)
	m := handleJob(logger, decoder)
	result := crackPassword(m)
	sendJobResults(logger, encoder, result)
	sendTermination(logger, encoder)
}
