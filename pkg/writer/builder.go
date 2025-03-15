package writer

import (
	"fmt"

	"github.com/PsychoPunkSage/ErgoFS/pkg/types"
	"github.com/PsychoPunkSage/ErgoFS/pkg/util"
)

// WriteSuperblock writes the superblock to the filesystem image
func WriteSuperblock(sbi *types.SuperBlkInfo, sbBh *types.BufferHead, blocks *uint32) error {
	types.Debug(types.EROFS_DBG, "Writing superblock")

	// Map the buffer to get the block address
	*blocks = util.MapBh(sbi.Bmgr, nil)

	// Ensure TotalBlocks is at least the current tail block address
	// This is important as the tail block address represents the next block to allocate
	if sbi.TotalBlocks < uint64(sbi.Bmgr.TailBlkAddr) {
		sbi.TotalBlocks = uint64(sbi.Bmgr.TailBlkAddr)
	}
	// Add 1 for the superblock itself if it's not already counted
	if sbi.TotalBlocks == 0 {
		sbi.TotalBlocks = 1
	}
	types.Debug(types.EROFS_DBG, "Total blocks in filesystem: %d", sbi.TotalBlocks)

	// Generate the superblock buffer
	sbBuf, err := sbi.WriteSuperblock()
	if err != nil {
		return fmt.Errorf("failed to generate superblock: %v", err)
	}

	// Get the offset where to write the superblock
	var offset uint64
	if sbBh != nil {
		offset = util.BhTell(sbBh, false)
	}

	types.Debug(types.EROFS_DBG, "Superblock will be written at offset %d", offset)

	// Write the superblock to the device
	err = util.DevWrite(sbi, sbBuf, offset, uint64(types.EROFS_SUPER_END))
	if err != nil {
		return fmt.Errorf("failed to write superblock: %v", err)
	}

	types.Debug(types.EROFS_DBG, "Superblock written successfully")

	// Drop the buffer head if provided
	if sbBh != nil {
		util.BDrop(sbBh, false)
	}

	return nil
}

// EnableSuperblockChecksum enables and computes the superblock checksum
func EnableSuperblockChecksum(sbi *types.SuperBlkInfo) (uint32, error) {
	types.Debug(types.EROFS_DBG, "Enabling superblock checksum")

	// Read the first block
	buf := make([]byte, sbi.ErofsBlockSize())

	// TODO: Implement reading the superblock from disk
	// For now, let's generate a fresh superblock
	sbBuf, err := sbi.WriteSuperblock()
	if err != nil {
		return 0, fmt.Errorf("failed to generate superblock: %v", err)
	}

	copy(buf, sbBuf[:sbi.ErofsBlockSize()])

	// Compute and set the checksum
	crc, err := sbi.EnableSuperblockChecksum(buf)
	if err != nil {
		return 0, fmt.Errorf("failed to enable superblock checksum: %v", err)
	}

	types.Debug(types.EROFS_DBG, "Computed superblock checksum: 0x%08x", crc)

	// Write the updated superblock with checksum
	err = util.BlkWrite(sbi, buf, 0, 1)
	if err != nil {
		return 0, fmt.Errorf("failed to write checksummed superblock: %v", err)
	}

	types.Debug(types.EROFS_DBG, "Superblock with checksum written successfully")
	return crc, nil
}
