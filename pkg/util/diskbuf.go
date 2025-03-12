package util

import (
	"fmt"
	"io"
	"os"
	"syscall"
	"unsafe"

	"github.com/PsychoPunkSage/ErgoFS/pkg/types"
)

// DevWrite writes data to the device
func DevWrite(sbi *types.SuperBlkInfo, buf []byte, offset uint64, length uint64) error {
	// Debug info
	types.Debug(types.EROFS_DBG, "Writing %d bytes to device at offset %d", length, offset)

	// Open the device file
	file, err := os.OpenFile(sbi.DevName, os.O_WRONLY, 0)
	if err != nil {
		return fmt.Errorf("failed to open device %s: %v", sbi.DevName, err)
	}
	defer file.Close()

	// Seek to the offset
	_, err = file.Seek(int64(offset), io.SeekStart)
	if err != nil {
		return fmt.Errorf("failed to seek to offset %d: %v", offset, err)
	}

	// Write the data
	n, err := file.Write(buf[:length])
	if err != nil {
		return fmt.Errorf("failed to write data: %v", err)
	}

	if uint64(n) != length {
		return fmt.Errorf("wrote only %d of %d bytes", n, length)
	}

	types.Debug(types.EROFS_DBG, "Successfully wrote %d bytes", n)
	return nil
}

// DevResize resizes the device file
func DevResize(sbi *types.SuperBlkInfo, blocks uint32) error {
	size := uint64(blocks) << sbi.BlkSzBits

	types.Debug(types.EROFS_DBG, "Resizing device to %d blocks (%d bytes)", blocks, size)

	// Open the device file
	file, err := os.OpenFile(sbi.DevName, os.O_WRONLY, 0)
	if err != nil {
		return fmt.Errorf("failed to open device %s: %v", sbi.DevName, err)
	}
	defer file.Close()

	// Truncate the file to the desired size
	err = file.Truncate(int64(size))
	if err != nil {
		return fmt.Errorf("failed to resize device to %d bytes: %v", size, err)
	}

	types.Debug(types.EROFS_DBG, "Device successfully resized")
	return nil
}

// BlkWrite writes blocks to the device
func BlkWrite(sbi *types.SuperBlkInfo, buf []byte, blkAddr uint32, nblocks uint32) error {
	offset := uint64(blkAddr) << sbi.BlkSzBits
	length := uint64(nblocks) << sbi.BlkSzBits

	types.Debug(types.EROFS_DBG, "Writing %d blocks starting at block %d", nblocks, blkAddr)
	return DevWrite(sbi, buf, offset, length)
}

// DevOpen opens a device and sets up the SuperBlkInfo structure accordingly
// This function is equivalent to erofs_dev_open in the C implementation
func DevOpen(sbi *types.SuperBlkInfo, dev string, flags int) error {
	var (
		mode     os.FileMode
		again    bool
		fileInfo os.FileInfo
		err      error
		stat     syscall.Stat_t
		statfs   syscall.Statfs_t
	)

	ro := (flags & syscall.O_ACCMODE) == os.O_RDONLY
	truncate := (flags & os.O_TRUNC) != 0

	// Open the device/file with appropriate flags
	openFlags := os.O_RDONLY
	if !ro {
		openFlags = os.O_RDWR | os.O_CREATE
	}

	file, err := os.OpenFile(dev, openFlags, 0644)
	if err != nil {
		types.Error("failed to open %s: %s", dev, err.Error())
		return err
	}

	fd := int(file.Fd())

	if ro || !truncate {
		goto out
	}

	// Get file info
	fileInfo, err = file.Stat()
	if err != nil {
		types.Error("failed to stat(%s): %s", dev, err.Error())
		file.Close()
		return err
	}

	mode = fileInfo.Mode()

	switch {
	case mode&os.ModeDevice != 0:
		// Block device handling
		ret := GetBlockDeviceSize(fd, &sbi.Devsz)
		if ret != 0 {
			types.Error("failed to get block device size(%s): %s", dev, err.Error())
			file.Close()
			return fmt.Errorf("failed to get block device size: errno %d", -ret)
		}

		sbi.Devsz = roundDown(sbi.Devsz, sbi.ErofsBlockSize())

		ret = BDevDiscard(fd, 0, sbi.Devsz)
		if ret != 0 {
			types.Error("failed to erase block device(%s): %s", dev, err.Error())
			// Note: The C code continues despite this error
		}

	case mode.IsRegular():
		// Regular file handling
		if fileInfo.Size() > 0 {
		repeat:
			// Check filesystem type for EXT4 and BTRFS workaround
			if !again && syscall.Fstatfs(fd, &statfs) == nil {
				// EXT4 magic: 0xEF53, BTRFS magic: 0x9123683E
				if statfs.Type == 0xEF53 || statfs.Type == 0x9123683E {
					file.Close()
					os.Remove(dev)
					again = true

					// Reopen the file
					file, err = os.OpenFile(dev, openFlags, 0644)
					if err != nil {
						types.Error("failed to reopen %s: %s", dev, err.Error())
						return err
					}
					fd = int(file.Fd())
					goto repeat
				}
			}

			if again {
				file.Close()
				return fmt.Errorf("not empty")
			}

			// Truncate the file
			err = file.Truncate(0)
			if err != nil {
				types.Error("failed to truncate(%s).", dev)
				file.Close()
				return err
			}
		}

		// Get block size (this is OS-specific)
		if err := syscall.Fstat(fd, &stat); err != nil {
			file.Close()
			return err
		}
		sbi.DevBlkSz = int(stat.Blksize)

	default:
		types.Error("bad file type (%s, %o).", dev, mode)
		file.Close()
		return fmt.Errorf("invalid file type")
	}

out:
	// Store device name
	sbi.DevName = dev

	// In Go, we'll use the file object instead of just the fd
	// We'll need to add a field to SuperBlkInfo to store this
	sbi.BDev.Fd = fd

	types.Info("successfully opened %s", dev)
	return nil
}

// GetBlockDeviceSize gets the size of a block device
// Equivalent to erofs_get_bdev_size in C
func GetBlockDeviceSize(fd int, bytes *uint64) int {
	const (
		// BLKGETSIZE64 is the ioctl command for getting the size of a block device in bytes
		BLKGETSIZE64 = 0x80081272 // _IOR(0x12, 114, uint64)

		// BLKGETSIZE is the ioctl command for getting the size of a block device in sectors
		BLKGETSIZE = 0x1260 // _IO(0x12, 96)
	)

	// // Set initial error to ENOTSUP
	// syscall.Errno = syscall.ENOTSUP

	var size uint64
	_, _, errno := syscall.Syscall(syscall.SYS_IOCTL, uintptr(fd), BLKGETSIZE64, uintptr(unsafe.Pointer(bytes)))
	if errno != 0 {
		return 0
	}

	// Fall back to BLKGETSIZE if BLKGETSIZE64 fails
	_, _, errno = syscall.Syscall(syscall.SYS_IOCTL, uintptr(fd), BLKGETSIZE, uintptr(unsafe.Pointer(&size)))
	if errno == 0 {
		*bytes = uint64(size) << 9 // Convert sectors to bytes (sector size is 512 bytes)
		return 0
	}

	// If both ioctls fail, return the negative of errno
	return -int(syscall.ENOTSUP)
}

// BDevDiscard discards/trims a range on a block device
// Equivalent to erofs_bdev_discard in C
func BDevDiscard(fd int, block uint64, count uint64) int {
	const (
		// BLKDISCARD is the ioctl command for discarding blocks on a block device
		BLKDISCARD = 0x1277 // _IO(0x12, 119)
	)

	// Create a range array [block, count]
	rangeArr := [2]uint64{block, count}

	// Call the ioctl with BLKDISCARD
	_, _, errno := syscall.Syscall(
		syscall.SYS_IOCTL,
		uintptr(fd),
		BLKDISCARD,
		uintptr(unsafe.Pointer(&rangeArr)),
	)

	// Check for errors
	if errno != 0 {
		return -int(errno)
	}

	return 0
}

// roundDown rounds x down to the nearest multiple of y
func roundDown(x, y uint64) uint64 {
	return (x / y) * y
}
