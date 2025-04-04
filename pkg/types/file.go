package types

import (
	"syscall"
	"unsafe"
)

type ErofsVfops struct {
	// Function pointers are replaced with function types in Go
	Pread     func(vf *ErofsVFile, buf []byte, offset uint64, len uint64) int64
	Pwrite    func(vf *ErofsVFile, buf []byte, offset uint64, len uint64) int64
	Fsync     func(vf *ErofsVFile) int
	Fallocate func(vf *ErofsVFile, offset uint64, len uint64, pad bool) int
	Ftruncate func(vf *ErofsVFile, length uint64) int
	Read      func(vf *ErofsVFile, buf []byte, len uint64) int64
	Lseek     func(vf *ErofsVFile, offset uint64, whence int) int64
	Fstat     func(vf *ErofsVFile, buf *syscall.Stat_t) int
	Xcopy     func(vout *ErofsVFile, pos int64, vin *ErofsVFile, len uint, noseek bool) int
}

type ErofsVFile struct {
	Ops *ErofsVfops

	Offset uint64
	Fd     int

	// Payload provides alternative access to Offset and Fd as a byte array
	// Go doesn't have unions, so this is a common pattern to mimic them
	// using unsafe.Pointer to access the same memory
}

// GetPayload returns the payload byte array view of the file data
func (vf *ErofsVFile) GetPayload() [16]byte {
	var payload [16]byte
	// This is a way to access the same memory region as the Offset and Fd fields
	// It's similar to how C unions work
	data := (*[16]byte)(unsafe.Pointer(&vf.Offset))
	copy(payload[:], data[:])
	return payload
}

// SetPayload sets the file data using the payload byte array
func (vf *ErofsVFile) SetPayload(payload [16]byte) {
	// Copy the payload into the memory used by Offset and Fd
	data := (*[16]byte)(unsafe.Pointer(&vf.Offset))
	copy(data[:], payload[:])
}
