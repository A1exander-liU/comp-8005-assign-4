package main

import (
	"encoding/gob"
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"time"

	utils "github.com/A1exander-liU/comp-8005-assign-1"
	"go.uber.org/zap"
)

type settings struct {
	controllerIP      string
	controllerPort    int
	controllerAddress string
}

func crackPassword(message utils.Message) {
	log.Println(message)
	log.Println("cracking password")
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

	_ = encoder.Encode(utils.Message{Version: "1", Type: "STATUS", Message: "hello"})
	_ = encoder.Encode(utils.Message{Version: "1", Type: "STATUS", Message: "world"})
	time.Sleep(5 * time.Second)
	_ = encoder.Encode(utils.Message{Version: "1", Type: "DONE", Message: "hello"})
}
