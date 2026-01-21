package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"syscall"

	utils "github.com/A1exander-liU/comp-8005-assign-1"
)

type Controller struct {
	socket net.Listener
}

type settings struct {
	shadowfile string
	username   string
	port       int
}

func listen(address string) net.Listener {
	server, err := net.Listen("tcp", address)
	if err != nil {
		fmt.Println("Failed to listen:\n", err)
		os.Exit(1)
	}
	fmt.Printf("Server listening on %s\n", address)
	return server
}

func cleanup(conn net.Listener) {
	if conn != nil {
		err := conn.Close()
		if err != nil {
			log.Fatal("Close error:", err)
		}

		log.Println("Server closed successfully")
	}
	log.Println("Exiting")
	os.Exit(0)
}

func handleSigInt(channel chan os.Signal, exit func(net.Listener), conn net.Listener) {
	for {
		sig := <-channel

		switch sig {
		case os.Interrupt, syscall.SIGINT:
			exit(conn)
		}
	}
}

func setupServer(port int) net.Listener {
	address := fmt.Sprintf("[::]:%d", port)

	server, err := net.Listen("tcp", address)
	if err != nil {
		log.Fatal(err)
	}

	log.Println("Listening on:", server.Addr().String())
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
	settings := parseArguments()
	handleArguments(&settings)

	fmt.Println(settings)

	server := setupServer(settings.port)

	sigChan := make(chan os.Signal, 1)
	go handleSigInt(sigChan, cleanup, server)

	for {
		conn, err := server.Accept()
		if err != nil {
			log.Println("accept error:", err)
			continue
		}

		message := utils.Receive(conn)
		fmt.Println(message)
	}
}
