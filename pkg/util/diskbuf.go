package util

import (
	"fmt"
	"io"
	"os"

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
