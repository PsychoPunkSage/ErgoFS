package types

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"

	errs "github.com/PsychoPunkSage/ErgoFS/pkg/errors"
)

// BufferManager manages buffer blocks
type BufferManager struct {
	Sbi             *SuperBlkInfo
	MappedBuckets   [META + 1][EROFS_MAX_BLOCK_SIZE]ListHead
	BlkH            BufferBlock
	TailBlkAddr     uint32
	MetaBlkCnt      uint32
	LastMappedBlock *BufferBlock
}

// BufferBlock represents a buffer block
type BufferBlock struct {
	List       ListHead
	MappedList ListHead
	BlkAddr    uint32
	Type       int
	Buffers    BufferHead
}

// BufferHead represents a buffer head
type BufferHead struct {
	List      ListHead
	Block     *BufferBlock
	Off       uint64
	Op        *BufferHeadOps
	FsPrivate interface{}
}

// BufferHeadOps defines operations for buffer heads
type BufferHeadOps struct {
	Flush func(*BufferHead) int
}

type ErofsDeviceSlot struct {
	Tag           [64]byte
	Blocks        uint32
	MappedBlkAddr uint32
	Reserved      [56]byte
}

const EROFS_DEVT_SLOT_SIZE = 64 + 4 + 4 + 56

// SkipWriteOps defines operations that skip writing
var SkipWriteOps = &BufferHeadOps{
	Flush: func(bh *BufferHead) int {
		return -errs.EBUSY
	},
}

var DropDirectlyBhops = BufferHeadOps{
	Flush: func(bh *BufferHead) int {
		return BhFlushGenericEnd(bh)
	},
}

var SkipWriteBhops = BufferHeadOps{
	Flush: func(bh *BufferHead) int {
		return -errs.EBUSY
	},
}

// ErofsBufferInit initializes a buffer manager
func ErofsBufferInit(sbi *SuperBlkInfo, startblk uint32) *BufferManager {
	bufmgr := new(BufferManager)

	InitListHead(&bufmgr.BlkH.List)
	bufmgr.BlkH.BlkAddr = NULL_ADDR
	bufmgr.LastMappedBlock = &bufmgr.BlkH

	for i := range len(bufmgr.MappedBuckets) {
		for j := range len(bufmgr.MappedBuckets[0]) {
			InitListHead(&bufmgr.MappedBuckets[i][j])
		}
	}

	bufmgr.TailBlkAddr = startblk
	bufmgr.Sbi = sbi
	return bufmgr
}

// ReserveSuperblock reserves space for the superblock
func ReserveSuperblock(bmgr *BufferManager) (*BufferHead, error) {
	bh, err := Balloc(bmgr, META, 0, 0, 0)
	if err != nil {
		return nil, fmt.Errorf("failed to allocate super: %v", err)
	}

	bh.Op = SkipWriteOps
	errr := BhBalloon(bh, uint64(EROFS_SUPER_END))
	if errr < 0 {
		BDrop(bh, true)
		return nil, fmt.Errorf("failed to balloon super: %v", err)
	}

	// Make sure the superblock is at the beginning
	MapBh(nil, bh.Block)
	if BhTell(bh, false) != 0 {
		BDrop(bh, true)
		return nil, errors.New("failed to pin super block @ 0")
	}

	return bh, nil
}

// erofsWriteSb is the Go equivalent of erofs_writesb
func WriteSuperBlock(sbi *SuperBlkInfo, sbBh *BufferHead, blocks *uint32) int {
	// Create the superblock structure
	sb := SuperBlock{
		Magic:            (EROFS_SUPER_MAGIC_V1),
		BlkSzBits:        sbi.BlkSzBits,
		RootNid:          uint16(sbi.RootNid),
		Inos:             (sbi.Inos),
		BuildTime:        (sbi.BuildTime),
		BuildTimeNsec:    (sbi.BuildTimeNsec),
		MetaBlkAddr:      (sbi.MetaBlkAddr),
		XattrBlkAddr:     (sbi.XattrBlkAddr),
		XattrPrefixCount: sbi.XattrPrefixCount,
		XattrPrefixStart: (sbi.XattrPrefixStart),
		FeatureIncompat:  (sbi.FeatureIncompat),
		FeatureCompat:    (sbi.FeatureCompat & ^EROFS_FEATURE_COMPAT_SB_CHKSUM),
		ExtraDevices:     (sbi.ExtraDevices),
		DevtSlotOff:      (sbi.DevtSlotOff),
		PackedNid:        (sbi.PackedNid),
	}

	// Calculate rounded up block size
	sbBlksize := Round_Up(EROFS_SUPER_END, ErofsBlkSiz(sbi))

	// Get the blocks count
	*blocks = MapBh(sbi.Bmgr, nil)
	sb.Blocks = (*blocks)

	// Copy UUID and volume name
	copy(sb.UUID[:], sbi.UUID[:])
	copy(sb.VolumeName[:], sbi.VolumeName[:])

	// Set compression configuration
	if ErofsSbHasComprCfgs(sbi) {
		// sb.U1.AvailableComprAlgs = uint16ToLe16(sbi.AvailableComprAlgs)
		sb.CompressInfo = (sbi.AvailableComprAlgs)
	} else {
		// sb.U1.Lz4MaxDistance = uint16ToLe16(sbi.Lz4.MaxDistance)
		sb.CompressInfo = (sbi.Lz4.MaxDistance)
	}

	// Allocate memory for the superblock
	buf := make([]byte, sbBlksize)
	// if buf == nil {
	// 	// erofsErr("failed to allocate memory for sb: %s", erofsStrerror(-ENOMEM))
	// 	return -types.ENOMEM
	// }

	// PPS:: MY Method || Convert the in-memory superblock to on-disk format and copy to buffer
	diskSb := sb.ToDisk()
	// Create a byte slice from the diskSb struct
	var diskSbBytes bytes.Buffer
	binary.Write(&diskSbBytes, binary.LittleEndian, diskSb)
	// Copy the serialized data to the buffer at the appropriate offset
	copy(buf[EROFS_SUPER_OFFSET:], diskSbBytes.Bytes())

	// // Copy superblock data to the buffer at the appropriate offset
	// sbBytes := (*[unsafe.Sizeof(sb)]byte)(unsafe.Pointer(&sb))[:]
	// copy(buf[types.EROFS_SUPER_OFFSET:], sbBytes)

	// Calculate the write position
	var writePos uint64 = 0
	if sbBh != nil {
		writePos = BhTell(sbBh, false)
	}

	// Write to device
	ret := ErofsDevWrite(sbi, buf, writePos, int(EROFS_SUPER_END))

	// Clean up
	if sbBh != nil {
		BDrop(sbBh, false)
	}

	return ret
}
