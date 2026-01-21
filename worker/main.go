package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"os"

	utils "github.com/A1exander-liU/comp-8005-assign-1"
)

type settings struct {
	controllerIP   string
	controllerPort int
}

func setupServer(controllerIP string, controllerPort int) net.Conn {
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
}

func parseArguments() settings {
	var settings settings

	flag.StringVar(&settings.controllerIP, "c", "", "controller ip")
	flag.IntVar(&settings.controllerPort, "p", 0, "controller port number")

	flag.Parse()

	return settings
}

func main() {
	settings := parseArguments()
	handleArguments(&settings)

	fmt.Println(settings)

	socket := setupServer(settings.controllerIP, settings.controllerPort)

	utils.Send(socket, utils.Message{Version: "1", Type: "a", Message: "hello"})
}
