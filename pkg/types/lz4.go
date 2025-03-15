package types

import (

	// "github.com/PsychoPunkSage/ErgoFS/pkg/types"

	"github.com/pierrec/lz4" // Using the Go lz4 package
)

// Helper function for max of two uint16 values
func maxU16(a, b uint16) uint16 {
	if a > b {
		return a
	}
	return b
}

// Lz4CompressDestsize compresses src into dst with a destination size constraint
func Lz4CompressDestsize(c *ErofsCompress,
	src []byte, srcsize *uint,
	dst []byte, dstsize uint) int {
	srcSize := int(*srcsize)

	// Call to LZ4_compress_destSize adapted for Go
	// Using the lz4 package's equivalent functionality
	compressor := lz4.NewWriter(nil)

	// Set up a destination size limited writer
	// This is an approximation as Go's lz4 library doesn't have a direct
	// equivalent to LZ4_compress_destSize
	compressor.Reset(limitWriter{buf: dst, limit: int(dstsize)})

	n, err := compressor.Write(src[:srcSize])
	if err != nil {
		return -EFAULT
	}

	rc := compressor.Close()
	if rc != nil {
		return -EFAULT
	}

	// Update srcsize with how much we actually read
	*srcsize = uint(n)

	// Return compressed size
	return len(dst)
}

// LimitWriter is a writer that stops after a certain number of bytes
type limitWriter struct {
	buf   []byte
	pos   int
	limit int
}

func (w limitWriter) Write(p []byte) (n int, err error) {
	remaining := w.limit - w.pos
	if remaining <= 0 {
		return 0, nil
	}
	if len(p) > remaining {
		p = p[:remaining]
	}
	n = copy(w.buf[w.pos:], p)
	w.pos += n
	return n, nil
}

// CompressorLz4Exit cleans up the LZ4 compressor
func CompressorLz4Exit(c *ErofsCompress) int {
	return 0
}

// CompressorLz4Init initializes the LZ4 compressor
func CompressorLz4Init(c *ErofsCompress) int {
	c.Sbi.Lz4.MaxDistance = maxU16(c.Sbi.Lz4.MaxDistance, LZ4_DISTANCE_MAX)
	return 0
}

// ErofsCompressorLz4 defines the LZ4 compressor operations
var ErofsCompressorLz4 = ErofsCompressor{
	Init:             CompressorLz4Init,
	Exit:             CompressorLz4Exit,
	CompressDestSize: Lz4CompressDestsize,
}
