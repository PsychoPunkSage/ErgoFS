package types

import (
	"errors"
	"fmt"
	"syscall"
)

// ErofsBlkRead reads blocks from the device
func ErofsBlkRead(sbi *SuperBlkInfo, deviceID int, buf []byte, start, nblocks uint32) int {
	i, _ := ErofsDevRead(sbi, deviceID, buf, ErofsPos(sbi, uint64(start)), int64(ErofsPos(sbi, uint64(nblocks))))
	return int(i)
}

func ErofsBlkWrite(sbi *SuperBlkInfo, buf []byte, blkAddr, nblocks uint32) int {
	return ErofsDevWrite(sbi, buf, ErofsPos(sbi, uint64(blkAddr)), int(ErofsPos(sbi, uint64(nblocks))))
}

// ErofsDevResize resizes the device to the given number of blocks
func ErofsDevResize(sbi *SuperBlkInfo, blocks uint32) int {
	return ErofsIoFtruncate(sbi.BDev, uint64(blocks)*uint64(ErofsBlkSiz(sbi)))
}

// ErofsDevWrite writes data to an EROFS device
func ErofsDevWrite(sbi *SuperBlkInfo, buf []byte, offset uint64, length int) int {
	written, _ := ErofsIoPwrite(sbi.BDev, buf, offset, length)
	if written != length {
		return -EIO
	}
	return 0
}

// ErofsDevRead reads data from an EROFS device
func ErofsDevRead(sbi *SuperBlkInfo, deviceID int, buf []byte, offset uint64, length int64) (int64, error) {
	var read int64
	var err error

	if deviceID > 0 {
		if deviceID > int(sbi.NBlobs) {
			return 0, fmt.Errorf("invalid device id %d: %w", deviceID, syscall.Errno(syscall.EIO))
		}

		// Create a temporary ErofsVfile with the blob file descriptor
		vfile := ErofsVFile{
			Fd: int(sbi.BlobFd[deviceID-1]),
		}

		read, err = ErofsIoPread(&vfile, buf, offset, length)
	} else {
		// Read from the main device
		read, err = ErofsIoPread(sbi.BDev, buf, offset, length)
	}

	if err != nil {
		return read, err
	}

	if read < length {
		// Log that we've reached the end of the device
		fmt.Printf("reach EOF of device @ %d, padding with zeroes\n", offset)

		// Pad the rest of the buffer with zeros
		for i := read; i < length; i++ {
			buf[i] = 0
		}
	}

	return 0, nil
}

// ErofsDevFillzero - inline function to fill with zeros
func ErofsDevFillzero(sbi *SuperBlkInfo, offset uint64, length uint64, pad bool) int64 {
	return ErofsIoFallocate(sbi.BDev, offset, length, pad)
}

func ErofsIoPread(vf *ErofsVFile, buf []byte, pos uint64, length int64) (int64, error) {
	var totalRead int64

	if GCfg.DryRun {
		return 0, nil
	}

	// If vf has custom read operations, use it
	if vf.Ops != nil && vf.Ops.Pread != nil {
		return int64(vf.Ops.Pread(vf, buf, pos, uint64(length))), nil
	}

	// Adjust position based on file offset
	pos += vf.Offset

	for totalRead < int64(length) {
		n, err := syscall.Pread(int(vf.Fd), buf[totalRead:], int64(pos))
		if err != nil {
			if errors.Is(err, syscall.EINTR) {
				continue // Retry if interrupted
			}
			fmt.Printf("Failed to read: %v\n", err)
			return totalRead, err
		}
		if n == 0 {
			break // End of file
		}

		pos += uint64(n)
		totalRead += int64(n)
	}

	return totalRead, nil
}

// ErofsIoFallocate - allocate or zero-fill file space
func ErofsIoFallocate(vf *ErofsVFile, offset uint64, length uint64, zeroout bool) int64 {
	// Static zero buffer of maximum block size
	var zero [EROFS_MAX_BLOCK_SIZE]byte
	var ret int

	if GCfg.DryRun {
		return 0
	}

	if vf.Ops != nil {
		return int64(vf.Ops.Fallocate(vf, offset, length, zeroout))
	}

	// // Equivalent to the C preprocessor conditional
	// if !zeroout && syscall.Fallocate(vf.Fd, syscall.FALLOC_FL_PUNCH_HOLE|syscall.FALLOC_FL_KEEP_SIZE,
	// 	int64(offset+vf.Offset), int64(length)) >= 0 {
	// 	return 0
	// }

	for length > uint64(EROFS_MAX_BLOCK_SIZE) {
		ret, _ = ErofsIoPwrite(vf, zero[:], offset, int(EROFS_MAX_BLOCK_SIZE))
		if ret < 0 {
			return int64(ret)
		}
		length -= uint64(ret)
		offset += uint64(ret)
	}

	written, err := ErofsIoPwrite(vf, zero[:], offset, int(length))
	if err == nil && int64(written) == int64(length) {
		return 0
	} else {
		return -EIO
	}
}

// ErofsIoPwrite writes data to an EROFS virtual file
func ErofsIoPwrite(vf *ErofsVFile, buf []byte, pos uint64, length int) (int, error) {
	written := 0

	// Skip actual writing in dry run mode
	if GCfg.DryRun {
		return 0, nil
	}

	// Use vfile operations if available
	if vf.Ops != nil {
		return int(vf.Ops.Pwrite(vf, buf, pos, uint64(length))), nil
	}

	// Adjust position by the file offset
	pos += vf.Offset

	// Write in a loop until complete or error
	for written < length {
		// Use pwrite syscall to write at specific position
		n, err := syscall.Pwrite(vf.Fd, buf[written:length], int64(pos))
		if n <= 0 {
			if n == 0 {
				break
			}
			if err != nil && err != syscall.EINTR {
				// ErofsErr("failed to write: %v", err)
				return written, err
			}
			n = 0
		}
		buf = buf[n:]
		pos += uint64(n)
		written += n
	}

	return written, nil
}

// ErofsIoFtruncate truncates a virtual file to the given length
func ErofsIoFtruncate(vf *ErofsVFile, length uint64) int {
	if GCfg.DryRun {
		return 0
	}

	if vf.Ops != nil {
		return vf.Ops.Ftruncate(vf, length)
	}

	var stat syscall.Stat_t
	err := syscall.Fstat(vf.Fd, &stat)
	if err != nil {
		// ErofsErr("failed to fstat: %s", syscall.Errno(ret).Error())
		return -1
	}

	length += vf.Offset
	if (stat.Mode&syscall.S_IFMT) == syscall.S_IFBLK || uint64(stat.Size) == length {
		return 0
	}

	err = syscall.Ftruncate(vf.Fd, int64(length))
	if err != nil {
		return -1 // or another error code as per your logic
	}
	return 0
}
