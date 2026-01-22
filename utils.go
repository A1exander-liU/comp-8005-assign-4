// Package utils
//
// Provides common functions for both controller and worker nodes:
//
// - message send and receiving
package utils

import (
	"net"
	"strconv"
)

type Message struct {
	Version, Type, Message string
}

// ParseAddress builds an IP:Port string. An empty string is returned if parsing failed.
func ParseAddress(ip string, port int) string {
	if ip == "localhost" {
		ip = "127.0.0.1"
	}

	parsedIP := net.ParseIP(ip)

	if parsedIP == nil {
		return ""
	}

	return net.JoinHostPort(parsedIP.String(), strconv.Itoa(port))
}
