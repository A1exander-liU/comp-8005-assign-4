package shared

import (
	"errors"
	"fmt"
	"net"
	"strconv"
	"strings"

	"github.com/go-crypt/crypt"
)

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

// ParseHash extracts the algorithm, parameters, salt, and hash from a GNU/Linux
// style shadowfile format password.
//
// Paramters may be an empty string depending on the algorithm used.
func ParseHash(hash string) (ShadowData, error) {
	// cut out first $ to prevent empty string in split
	cleanedHash, _, _ := strings.Cut(hash, ":")

	// 3 sections: algo + salt + hash
	// 4 sections: algo + parameters + salt + hash
	sections := strings.Split(cleanedHash[1:], "$")

	switch len(sections) {
	case 3:
		return ShadowData{Algorithm: sections[0], Salt: sections[1], Hash: sections[2]}, nil
	case 4:
		return ShadowData{Algorithm: sections[0], Parameters: sections[1], Salt: sections[2], Hash: sections[3]}, nil
	default:
		return ShadowData{}, fmt.Errorf("failed to parse hash: %s", hash)
	}
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
	characterSet := "abcdefgh"
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
