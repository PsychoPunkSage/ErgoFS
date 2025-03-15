package types

import "syscall"

// ErofsDevWrite writes data to an EROFS device
func ErofsDevWrite(sbi *SuperBlkInfo, buf []byte, offset uint64, length int) int {
	written, _ := ErofsIoPwrite(sbi.BDev, buf, offset, length)
	if written != length {
		return -EIO
	}
	return 0
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
