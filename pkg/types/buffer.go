package types

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
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

// SkipWriteOps defines operations that skip writing
var SkipWriteOps = &BufferHeadOps{
	Flush: func(bh *BufferHead) int {
		return -EBUSY
	},
}

var DropDirectlyBhops = BufferHeadOps{
	Flush: func(bh *BufferHead) int {
		return BhFlushGenericEnd(bh)
	},
}

var SkipWriteBhops = BufferHeadOps{
	Flush: func(bh *BufferHead) int {
		return -EBUSY
	},
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
	if sbi.ErofsSbHasComprCfgs() {
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

func ErofsBflush(bmgr *BufferManager, bb *BufferBlock) int {
	// fmt.Println("1")
	sbi := bmgr.Sbi
	blksiz := ErofsBlkSiz(sbi)
	// fmt.Println("2")

	// Check if list is empty
	if IsListEmpty(&bmgr.BlkH.List) {
		return 0
	}

	// Start with the first buffer block
	p := BufferBlockFromList(bmgr.BlkH.List.Next)
	if p == nil {
		return 0
	}

	for p != nil && &p.List != &bmgr.BlkH.List {
		// fmt.Println("3")

		// Save next before potentially freeing p
		var n *BufferBlock
		if p.List.Next != nil && p.List.Next != &bmgr.BlkH.List {
			n = BufferBlockFromList(p.List.Next)
		}

		// Exit if we hit the specified block
		if p == bb {
			break
		}

		blkaddr := MapBhInternal(p)

		// Process buffer heads - DO NOT use same List/Next comparison as outer loop
		skip := false

		// Check if the buffer list is empty
		if !IsListEmpty(&p.Buffers.List) {
			var bh *BufferHead
			// Get the first buffer head in the list
			bh = BufferHeadFromList(p.Buffers.List.Next)

			// Process all buffer heads in the list properly
			for bh != nil && &bh.List != &p.Buffers.List {
				// Save next before potentially removing bh
				var nbh *BufferHead
				if bh.List.Next != nil && bh.List.Next != &p.Buffers.List {
					nbh = BufferHeadFromList(bh.List.Next)
				}

				if bh.Op == SkipWriteOps {
					skip = true
				} else if bh.Op != nil {
					// Flush and remove bh
					ret := bh.Op.Flush(bh)
					if ret < 0 {
						return ret
					}
				}

				// Move to next or break
				if nbh == nil || nbh == bh {
					break
				}
				bh = nbh
			}
		}

		if !skip {
			padding := uint64(blksiz) - (p.Buffers.Off & (uint64(blksiz) - 1))
			if padding != uint64(blksiz) {
				ErofsDevFillzero(sbi, ErofsPos(sbi, blkaddr)-padding, padding, true)
			}

			if p.Type != DATA {
				bmgr.MetaBlkCnt += uint32(BlkRoundUp(sbi, p.Buffers.Off))
			}

			ErofsBfree(p)
		}

		// Move to next or break
		if n == nil || n == p {
			break
		}
		p = n
	}

	fmt.Println("Erofs Bflush successfully executed")
	return 0
}

// Helper function to get alignment size
func GetAlignSize(sbi *SuperBlkInfo, bufType int) (int, int) {
	if bufType == DATA {
		return int(sbi.ErofsBlockSize()), bufType
	}

	if bufType == INODE {
		return 32, // Size of struct erofs_inode_compact <PPS::> Need to code it>
			META
	} else if bufType == DIRA {
		return int(sbi.ErofsBlockSize()), META
	} else if bufType == XATTR {
		return 4, // Size of struct erofs_xattr_entry <PPS:: need to implement it>
			META
	} else if bufType == DEVT {
		return int(EROFS_DEVT_SLOT_SIZE), META
	}

	if bufType == META {
		return 1, bufType
	}

	return -EINVAL, 0 // Error
}

// Helper function to allocate a buffer block
func BlkAllocBuf(bmgr *BufferManager, aType int) (*BufferBlock, error) {
	bb := &BufferBlock{
		BlkAddr: NULL_ADDR,
		Type:    aType,
	}

	// Initialize lists
	InitListHead(&bb.List)
	InitListHead(&bb.MappedList)
	InitListHead(&bb.Buffers.List)

	bb.Buffers.Off = uint64(bmgr.Sbi.ErofsBlockSize())
	bb.Buffers.FsPrivate = bmgr

	// Add to block list
	ListAddTail(&bb.List, &bmgr.BlkH.List)

	return bb, nil
}

// Balloc allocates a new buffer head
func Balloc(bmgr *BufferManager, bufType int, size uint64, requiredExt, inlineExt uint32) (*BufferHead, error) {
	var bb *BufferBlock
	var bh *BufferHead
	var alignSize uint32
	var ret int

	ret, bufType = GetAlignSize(bmgr.Sbi, bufType)
	if ret < 0 {
		return nil, fmt.Errorf("invalid buffer type: %d", bufType)
	}

	if bufType < 0 || bufType > META {
		return nil, errors.New("invalid buffer type")
	}

	alignSize = uint32(ret)

	// Try to find if we could reuse an allocated buffer block
	bb, err := BfindForAttach(bmgr, bufType, size, requiredExt, inlineExt, alignSize)
	if err != nil {
		return nil, err
	}

	if bb != nil {
		// Reuse existing buffer block
		bh = new(BufferHead)
		// if bh == nil {
		// 	return nil, errors.New(fmt.Sprintf("ENOMEM: %d", ENOMEM))
		// }
	} else {
		// Get a new buffer block instead
		bb = new(BufferBlock)
		// if bb == nil {
		// 	return nil, errors.New(fmt.Sprintf("ENOMEM: %d", ENOMEM))
		// }

		bb.Type = bufType
		bb.BlkAddr = NULL_ADDR
		bb.Buffers.Off = 0
		bb.Buffers.FsPrivate = bmgr
		InitListHead(&bb.Buffers.List)

		if bufType == DATA {
			ListAdd(&bb.List, &bmgr.LastMappedBlock.List)
		} else {
			ListAddTail(&bb.List, &bmgr.BlkH.List)
		}

		InitListHead(&bb.MappedList)

		bh = new(BufferHead)
		// if bh == nil {
		// 	// No need to free the buffer block in Go as it will be garbage collected
		// 	return nil, errors.New(fmt.Sprintf("ENOMEM: %d", ENOMEM))
		// }
	}

	// Total required extensions
	totalRequiredExt := requiredExt + inlineExt

	// Look for an existing buffer block with enough space
	current := bmgr.BlkH.List.Next
	for current != &bmgr.BlkH.List {
		blockInterface := ContainerOf1(current, &BufferBlock{}, "List")
		block, ok := blockInterface.(*BufferBlock)
		if !ok {
			current = current.Next
			continue
		}

		// Skip if type doesn't match
		if block.Type != bufType {
			current = current.Next
			continue
		}

		if block.Buffers.Off >= size+uint64(totalRequiredExt)*uint64(alignSize) {
			bb = block
			break
		}
		current = current.Next
	}

	// No available buffer, so allocate a new one
	if bb == nil {
		var err error
		bb, err = BlkAllocBuf(bmgr, bufType)
		if err != nil {
			return nil, err
		}
	}

	if size > bb.Buffers.Off {
		return nil, fmt.Errorf("empty buffer block, there should exist at least one buffer head in a buffer block")
	}

	// Create a new buffer head
	bh = &BufferHead{
		Op:        nil,
		Block:     bb,
		Off:       size,
		FsPrivate: nil,
	}

	// Add to buffer list
	ListAddTail(&bh.List, &bb.Buffers.List)
	return bh, nil
}

// Battach attaches a new buffer head to an existing one
func Battach(bh *BufferHead, bufType int, size uint32) (*BufferHead, int) {
	bb := bh.Block
	bmgr := bb.Buffers.FsPrivate.(*BufferManager)

	// Get alignment size based on buffer type
	alignsize, _ := GetAlignSize(bmgr.Sbi, bufType)

	// Should be the tail bh in the corresponding buffer block
	if bh.List.Next != &bb.Buffers.List {
		return nil, -EINVAL
	}

	// Allocate new buffer head
	nbh := new(BufferHead)
	// if nbh == nil {
	// 	return nil, -ENOMEM
	// }

	// Attach the new buffer head
	err := BattachInternal(bb, nbh, uint64(size), uint32(alignsize), 0, false)
	if err < 0 {
		return nil, err
	}

	return nbh, 0
}

// BhBalloon expands a buffer head
func BhBalloon(bh *BufferHead, incr uint64) int {
	block := bh.Block
	if bh.List.Next != &block.Buffers.List {
		return -EINVAL
	}
	return BattachInternal(block, bh, incr, 1, 0, false)
}

// MapBh maps a buffer block
func MapBh(bmgr *BufferManager, bb *BufferBlock) uint32 {
	var t *BufferBlock

	if bmgr == nil && bb != nil {
		bmgr = bb.Buffers.FsPrivate.(*BufferManager)
	}
	t = bmgr.LastMappedBlock

	if bb != nil && bb.BlkAddr != NULL_ADDR {
		return bb.BlkAddr
	}

	for {
		nextEntry := t.List.Next
		nextInterface := ContainerOf1(nextEntry, &BufferBlock{}, "List")
		t, _ = nextInterface.(*BufferBlock)

		if t == &bmgr.BlkH {
			break
		}

		if t.BlkAddr != NULL_ADDR {
			panic("BUG: t.BlkAddr != NULL_ADDR")
		}

		MapBhInternal(t)

		if t == bb {
			break
		}
	}
	return bmgr.TailBlkAddr
}

// MapBhInternal is the internal implementation for mapping a buffer block (equivalent to __erofs_mapbh)
func MapBhInternal(bb *BufferBlock) uint64 {
	bmgr := bb.Buffers.FsPrivate.(*BufferManager)
	var blkaddr uint64

	if bb.BlkAddr == NULL_ADDR {
		bb.BlkAddr = bmgr.TailBlkAddr
		bmgr.LastMappedBlock = bb
		BupdateMapped(bb)
	}

	blkaddr = uint64(bb.BlkAddr) + BlkRoundUp(bmgr.Sbi, bb.Buffers.Off)
	if blkaddr > uint64(bmgr.TailBlkAddr) {
		bmgr.TailBlkAddr = uint32(blkaddr)

	}

	return blkaddr
}

// BhTell returns the offset of a buffer head
func BhTell(bh *BufferHead, end bool) uint64 {
	var offset uint64
	bb := bh.Block
	bmgr := bb.Buffers.FsPrivate.(*BufferManager)

	if bb.BlkAddr == NULL_ADDR { // NULL_ADDR
		return NULL_ADDR_UL
	}

	if end {
		// Get the next buffer head in the list
		nextEntry := bh.List.Next
		nextInterface := ContainerOf(nextEntry, &BufferHead{}, "List")
		nextBh, _ := nextInterface.(*BufferHead)
		offset = nextBh.Off
	} else {
		offset = bh.Off
	}

	return ErofsPos(bmgr.Sbi, uint64(bb.BlkAddr)) + offset
}

// BDrop drops a buffer head
func BDrop(bh *BufferHead, tryRevoke bool) {
	bb := bh.Block
	bmgr := bb.Buffers.FsPrivate.(*BufferManager)
	sbi := bmgr.Sbi
	blkaddr := bh.Block.BlkAddr
	rollback := false

	// tailBlkaddr could be rolled back after revoking all bhs
	if tryRevoke && blkaddr != NULL_ADDR &&
		bmgr.TailBlkAddr == blkaddr+uint32(BlkRoundUp(sbi, bb.Buffers.Off)) {
		rollback = true
	}

	bh.Op = &DropDirectlyBhops
	BhFlushGenericEnd(bh)

	if !IsListEmpty(&bb.Buffers.List) {
		return
	}

	if !rollback && bb.Type != DATA {
		bmgr.MetaBlkCnt += uint32(BlkRoundUp(sbi, bb.Buffers.Off))
	}
	ErofsBfree(bb)
	if rollback {
		bmgr.TailBlkAddr = blkaddr
	}
}
