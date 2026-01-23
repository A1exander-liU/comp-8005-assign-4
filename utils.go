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

type ShadowData struct {
	Algorithm, Salt, Hash string
}

type Message struct {
	Version, Type, Message string
	Data                   ShadowData
	Result                 string
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
