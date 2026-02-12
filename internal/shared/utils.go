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

func checkPassword(decoder *crypt.Decoder, plaintext, hash string) (bool, error) {
	digest, err := decoder.Decode(hash)
	if err != nil {
		return false, err
	}
	return digest.Match(plaintext), nil
}

func PartitionArray(array []string, count int) [][]string {
	partitions := [][]string{}
	arrayLength := len(array)

	partitionSize := arrayLength / count

	for i := range count {
		start := i * partitionSize
		end := start + partitionSize
		partitions = append(partitions, array[start:end])
	}

	num := 0
	for i := count * partitionSize; i < arrayLength; i++ {
		partitions[num%count] = append(partitions[num%count], array[i])
		num += 1
	}

	return partitions
}

// GenerateCandidatePasswords creates all possible passwords of the given length
// and character set.
func GenerateCandidatePasswords(characterSet string, length int) []string {
	candidates := []string{}

	characterSetLength := len(characterSet)

	total := 1

	for range length {
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
		candidates = append(candidates, testPassword)
	}

	return candidates
}

// CrackPassword tries all permutations of passwords up to `maxLength`. Returns
// the plaintext if crack was successful otherwise it will return an error.
//
// The total characte set includes:
//
// - alphanumeric characters (a-z, A-Z, 0-9)
//
// - special characters: @ # % ^ & * ( ) _ + - = . , : ; ?
func CrackPassword(decoder *crypt.Decoder, hash string, characterSet string, length int) (string, error) {
	characterSetLength := len(characterSet)

	total := 1

	for range length {
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
		cracked, err := checkPassword(decoder, testPassword, hash)
		if err != nil {
			return "", errors.New("failed to crack the password")
		}
		if cracked {
			return testPassword, nil
		}
	}

	return "", errors.New("failed to crack the password")
}
