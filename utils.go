// Package utils
//
// Provides common functions for both controller and worker nodes:
//
// - message send and receiving
package utils

import (
	"errors"
	"net"
	"strconv"

	"github.com/go-crypt/crypt"
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

func checkPassword(decoder *crypt.Decoder, plaintext, hash string) bool {
	digest, _ := decoder.Decode(hash)
	return digest.Match(plaintext)
}

// CrackPassword tries all permutations of passwords up to `maxLength`. Returns
// the plaintext if crack was successful otherwise it will return an error.
//
// The total characte set includes:
//
// - alphanumeric characters (a-z, A-Z, 0-9)
//
// - special characters: @ # % ^ & * ( ) _ + - = . , : ; ?
func CrackPassword(decoder *crypt.Decoder, hash string, maxLength int) (string, error) {
	characterSet := "abcd"
	characterSetLength := len(characterSet)

	for length := 1; length <= maxLength; length++ {
		total := 1

		for i := 0; i < length; i++ {
			total *= characterSetLength
		}

		for i := 0; i < total; i++ {
			combination := make([]byte, length)
			num := i

			for pos := length - 1; pos >= 0; pos-- {
				combination[pos] = characterSet[num%characterSetLength]
				num /= characterSetLength
			}

			testPassword := string(combination)
			if checkPassword(decoder, testPassword, hash) {
				return testPassword, nil
			}
		}
	}

	return "", errors.New("failed to crack the password")
}
