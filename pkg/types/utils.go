package types

import (
	"encoding/binary"
	"fmt"
	"os"
	"path/filepath"
	"unsafe"
)

// roundMask returns the mask for rounding operations
func RoundMask(x, y uint32) uint32 {
	return y - 1
}

// roundUp rounds x up to the nearest multiple of y
func Round_Up(x, y uint32) uint32 {
	return ((x - 1) | RoundMask(x, y)) + 1
}

// Roundup rounds up a number to the nearest multiple of align
func RoundUp(x, y uint64) uint64 {
	if y == 0 {
		return x // Avoid division by zero
	}
	return ((x + (y - 1)) / y) * y
}

func RoundDown(x, y int) int {
	return x - (x % y)
}

// Cmpsgn compares two numbers and returns sign (negative, zero, positive)
func Cmpsgn(a, b uint64) int {
	if a < b {
		return -1
	} else if a > b {
		return 1
	}
	return 0
}

// BlkRoundUp rounds up offset to the next block
func BlkRoundUp(sbi *SuperBlkInfo, offset uint64) uint64 {
	blksz := ErofsBlkSiz(sbi)
	return RoundUp(offset, uint64(blksz)) >> sbi.BlkSzBits
}

// roundDown rounds x down to the nearest multiple of y
func Round_Down(x, y uint32) uint32 {
	return x &^ RoundMask(x, y) // &^ is bitwise AND NOT in Go
}

// erofsBlknr computes the block number from an address
func ErofsBlknr(sbi *SuperBlkInfo, addr uint) uint {
	return addr >> sbi.BlkSzBits
}

// Le32ToCpu converts a little-endian uint32 to the CPU's native endianness
// This is the Go equivalent of the C macro le32_to_cpu
func Le32ToCpu(value uint32) uint32 {
	// Create a temporary buffer
	var buf [4]byte

	// Copy the value bytes to the buffer
	*(*uint32)(unsafe.Pointer(&buf[0])) = value

	// Read using little-endian byte order
	return binary.LittleEndian.Uint32(buf[:])
}

// CpuToLe32 converts a uint32 from CPU's native endianness to little-endian
// This is the Go equivalent of the C macro cpu_to_le32
func CpuToLe32(value uint32) uint32 {
	// Create a temporary buffer
	var buf [4]byte

	// Write using little-endian byte order
	binary.LittleEndian.PutUint32(buf[:], value)

	// Copy the buffer bytes back to a uint32
	return *(*uint32)(unsafe.Pointer(&buf[0]))
}

func ShowProgs(args []string) {
	if GCfg.DebugLevel >= EROFS_WARN {
		programName := filepath.Base(args[0])
		fmt.Printf("%s %s\n", programName, GCfg.Version)
	}
}

// isatty determines if stdout is a TTY (similar to C's isatty function)
func isatty() bool {
	fileInfo, err := os.Stdout.Stat()
	if err != nil {
		return false
	}

	// Check if it's a character device (TTY)
	// This is OS-specific but works on most Unix-like systems
	return (fileInfo.Mode() & os.ModeCharDevice) != 0
}
