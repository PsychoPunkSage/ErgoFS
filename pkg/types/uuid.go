package types

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"math/big"
	"strings"
	"syscall"

	errs "github.com/PsychoPunkSage/ErgoFS/pkg/errors"
	"golang.org/x/sys/unix"
)

var erofsGrndFlag uint = GRND_INSECURE

func getRandomNumber(out []byte, insecure bool) error {
	size := len(out)
	kflags := erofsGrndFlag
	flags := uint(0)
	if insecure {
		flags = uint(kflags)
	}

	for {
		var n int
		var err error

		// Try to use getrandom syscall if available
		n, err = unix.Getrandom(out, int(flags))

		if n == size {
			return nil
		}

		if err != syscall.EINTR {
			if err == syscall.ENOSYS && insecure {
				// Fall back to math/rand for insecure random if syscall not supported
				for i := 0; i < size; i++ {
					num, randErr := mathRand(0, 255)
					if randErr != nil {
						return randErr
					}
					out[i] = byte(num)
				}
				return nil
			} else if err == syscall.EINVAL && kflags != 0 {
				// Kernel likely does not support GRND_INSECURE
				erofsGrndFlag = 0
				kflags = 0
				continue
			}
			return err
		}
		// On EINTR, retry
	}
}

// mathRand generates a random number in the range [min, max]
func mathRand(min, max int64) (int64, error) {
	// Generate a cryptographically secure random number
	// This is different from the C implementation that uses rand(),
	// but provides better security even in fallback mode
	n, err := rand.Int(rand.Reader, big.NewInt(max-min+1))
	if err != nil {
		return 0, err
	}
	return n.Int64() + min, nil
}

func UUIDGenerate(out []byte) error {
	if len(out) < 16 {
		return errors.New("output buffer too short")
	}

	// create new UUID
	newUUID := make([]byte, 16)

	// Get Random bytes
	err := getRandomNumber(newUUID, true)
	if err != nil {
		return err
	}

	// set UUID version and variant bits (version 4, variant 1)
	newUUID[6] = (newUUID[6] & 0x0f) | 0x40 // version 4
	newUUID[8] = (newUUID[8] & 0x3f) | 0x80 // variant 1

	// copy to output
	copy(out, newUUID)
	return nil
}

func UuidParse(in string, uu []byte) int {
	if len(uu) < 16 {
		return -errs.EINVAL
	}

	// Remove all hyphens and spaces
	cleaned := strings.ReplaceAll(in, "-", "")
	cleaned = strings.ReplaceAll(cleaned, " ", "")

	// Check length
	if len(cleaned) != 32 {
		return -errs.EINVAL
	}

	// Decode hex string
	dst := make([]byte, hex.DecodedLen(len(cleaned)))
	_, err := hex.Decode(dst, []byte(cleaned))
	if err != nil {
		return -errs.EINVAL
	}

	// Copy to output
	copy(uu, dst)
	return 0
}
