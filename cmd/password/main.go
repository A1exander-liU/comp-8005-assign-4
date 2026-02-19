package main

import (
	"fmt"
	"slices"

	"github.com/A1exander-liU/comp-8005-assign-2/internal/shared"
)

// CartesianStrings returns all strings of length 1..maxSize built from chars.
// Example: chars="abc", maxSize=2 => a b c aa ab ac ba bb bc ca cb cc
func CartesianStrings(chars string, maxSize int) []string {
	if maxSize <= 0 || len(chars) == 0 {
		return nil
	}

	alphabet := []rune(chars)
	base := len(alphabet)

	out := make([]string, 0)

	// Generate for each length L = 1..maxSize
	for L := 1; L <= maxSize; L++ {
		// indices represent a number in base `base` with L digits
		indices := make([]int, L) // starts all zeros => "aaa..." for length L

		for {
			// Build current string
			runes := make([]rune, L)
			for i := 0; i < L; i++ {
				runes[i] = alphabet[indices[i]]
			}
			out = append(out, string(runes))

			// Increment base-N counter (right to left)
			pos := L - 1
			for pos >= 0 && indices[pos] == base-1 {
				indices[pos] = 0
				pos--
			}
			if pos < 0 {
				break // overflow => done for this length
			}
			indices[pos]++
		}
	}

	return out
}

// GenPassInRange creates all passwords in start and end
// a, b, c, ..., A, B, C, ..., 0, 1, 2, ..., :, ;, ?
func GenPassInRange(start string, end string, searchSpace string) {
	// reach end of current length start new
	// end of current length is when password equals to all '?'
	// length 1 = ?, 2 = ??, 3 = ??? and so on
	//
	// represent

	// [ ..., 7 8 9 @ # % ^ & * ( ) _ + - = . , : ; ? ] [ aa ab ac ad ae af ag ah ai aj ak al am an ao ap aq ar as at au av, ... ]
	//    1                                         78    79
	//    floor(80 / 79) = 1
	//    80 % 79 = 1
	//    1 means 1 full ?
	//    remainder 1 means character a
	//
	//    floor(540 / 79) =

	// need to account that this is base 79, each position adds 79 times more
	// 2 digits is 79 * 79 passwords
	// 6241 (0 - 6240)
	passwords := CartesianStrings(searchSpace, 3)
	fmt.Println(len(passwords))
	fmt.Println(len(searchSpace))
	fmt.Println()

	// ef = 400
	// e: 4, f: 5
	// eg = 401
	// e: 4, g: 6
	// num of passwords exponentially increases by each digit
	// can't just divide b
	//
	// 78 = ?
	// 79 = aa
	// [79] -> [79, 79, ..., 79]
	// 1: each starting digit has additional 79^0
	// 2: each starting digit has additional 79^1 - 1
	// 3: each starting digit has additional 79^2 - 1
	// and so on...
	// use base 79 as the representation
	// convert to base 79 to get the password
	// becomes efficient
	// just store start as int
	// start + chunk = end
	// can convert start and end to base 79

	// val += 6240

	val := 1000
	n := len(searchSpace)

	// determine place in base 79

	divisor := val
	result := []byte{}

	// 79
	//
	// 79 // 79 = 1
	// 79 % 79 = 0  -> A
	//
	// 1 // 79 = 0
	// 1 % 79 =
	// a

	digit := 0
	for {
		quotient := divisor / n
		remainder := divisor % n

		if digit > 0 {
			remainder -= 1
		}
		result = append(result, searchSpace[remainder])

		fmt.Println(quotient)
		if quotient == 0 {
			break
		}
		divisor = quotient
		digit += 1
	}
	slices.Reverse(result)

	fmt.Println("result:", string(result))
	fmt.Println(passwords[val])
}

func FromID(val int) string {
	// Essentially doing decimal conversion to base 79 since there are 79 different characters in search space.
	//
	// Does a -1 on digits after the first position
	// so that 79 becomes 'aa' instead of 'ab'
	n := len(shared.SearchSpace)
	result := []byte{}
	divisor := val
	digit := 0

	for {
		quotient := divisor / n
		remainder := divisor % n

		if digit > 0 {
			remainder -= 1
		}
		result = append(result, shared.SearchSpace[remainder])

		if quotient == 0 {
			break
		}
		divisor = quotient
		digit += 1
	}
	slices.Reverse(result)

	return string(result)
}

func GeneratePasswordsRange(start, end string) []string {
	passwords := []string{}

	return passwords
}

func main() {
	start := 0
	chunkSize := 1000
	fmt.Println(FromID(start), FromID(start+chunkSize))
}
