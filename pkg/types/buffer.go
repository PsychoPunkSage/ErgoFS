package types

import (
	"errors"
	"fmt"
	"reflect"
	"unsafe"
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
		// Implementation to skip writing
		return 0
	},
}

// ReserveSuperblock reserves space for the superblock
func ReserveSuperblock(bmgr *BufferManager) (*BufferHead, error) {
	bh, err := Balloc(bmgr, META, 0, 0, 0)
	if err != nil {
		return nil, fmt.Errorf("failed to allocate super: %v", err)
	}

	bh.Op = SkipWriteOps
	err = BhBalloon(bh, uint64(EROFS_SUPER_END))
	if err != nil {
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

// BfindForAttach finds a buffer block to attach to
func BfindForAttach(bmgr *BufferManager, btype int, size uint64,
	requiredExt, inlineExt, alignsize uint32) (*BufferBlock, error) {

	blkSiz := ErofsBlkSiz(bmgr.Sbi)
	var cur, bb *BufferBlock
	var used0, usedBefore, usedmax, used uint32
	var ret int

	used0 = ((uint32(size) + requiredExt) & (blkSiz - 1)) + inlineExt
	// inline data should be in the same fs block
	if used0 > blkSiz {
		return nil, fmt.Errorf("ENOSPC: %d", ENOSPC)
	}

	if used0 == 0 || alignsize == blkSiz {
		return nil, nil
	}

	usedmax = 0
	bb = nil

	// try to find a most-fit mapped buffer block first
	if uint32(size)+requiredExt+inlineExt >= blkSiz {
		goto skipMapped
	}

	usedBefore = uint32(RoundDown(int(blkSiz-(uint32(size)+requiredExt+inlineExt)), int(alignsize)))
	for ; usedBefore > 0; usedBefore-- {
		bt := &bmgr.MappedBuckets[btype][usedBefore]

		if IsListEmpty(bt) {
			continue
		}

		curEntry := bt.Next
		curInterface := ContainerOf(curEntry, &BufferBlock{}, "MappedList")
		cur, _ = curInterface.(*BufferBlock)

		// last mapped block can be expended, don't handle it here
		nextEntry := cur.List.Next
		nextInterface := ContainerOf(nextEntry, &BufferBlock{}, "List")
		next, _ := nextInterface.(*BufferBlock)

		if next.BlkAddr == NULL_ADDR {
			if cur != bmgr.LastMappedBlock {
				panic("BUG: cur != bmgr.LastMappedBlock")
			}
			continue
		}

		if cur.Type != btype {
			panic("BUG: cur.Type != btype")
		}
		if cur.BlkAddr == NULL_ADDR {
			panic("BUG: cur.BlkAddr == NULL_ADDR")
		}
		if usedBefore != uint32(cur.Buffers.Off&uint64(blkSiz-1)) {
			panic("BUG: usedBefore != (cur.Buffers.Off & (blksiz - 1))")
		}

		ret = BattachInternalExtend(cur, nil, size, alignsize, requiredExt+inlineExt, true)
		if ret < 0 {
			// panic("BUG: ret < 0")
			continue
		}

		// should contain all data in the current block
		used = uint32(ret) + requiredExt + inlineExt
		if used > blkSiz {
			panic("BUG: used > blksiz")
		}

		bb = cur
		usedmax = used
		break
	}

skipMapped:
	// try to start from the last mapped one, which can be expended
	cur = bmgr.LastMappedBlock
	if cur == &bmgr.BlkH {
		nextEntry := cur.List.Next
		nextInterface := ContainerOf(nextEntry, &BufferBlock{}, "List")
		cur, _ = nextInterface.(*BufferBlock)
	}

	for cur != &bmgr.BlkH {
		usedBefore = uint32(cur.Buffers.Off & uint64(blkSiz-1))

		// skip if buffer block is just full
		if usedBefore == 0 {
			nextEntry := cur.List.Next
			nextInterface := ContainerOf(nextEntry, &BufferBlock{}, "List")
			cur, _ = nextInterface.(*BufferBlock)
			continue
		}

		// skip if the entry which has different type
		if cur.Type != btype {
			nextEntry := cur.List.Next
			nextInterface := ContainerOf(nextEntry, &BufferBlock{}, "List")
			cur, _ = nextInterface.(*BufferBlock)
			continue
		}

		ret = BattachInternalExtend(cur, nil, size, alignsize, requiredExt+inlineExt, true)
		if ret < 0 {
			nextEntry := cur.List.Next
			nextInterface := ContainerOf(nextEntry, &BufferBlock{}, "List")
			cur, _ = nextInterface.(*BufferBlock)
			continue
		}

		used = ((uint32(ret) + requiredExt) & (blkSiz - 1)) + inlineExt

		// should contain inline data in current block
		if used > blkSiz {
			nextEntry := cur.List.Next
			nextInterface := ContainerOf(nextEntry, &BufferBlock{}, "List")
			cur, _ = nextInterface.(*BufferBlock)
			continue
		}

		/*
		 * remaining should be smaller than before or
		 * larger than allocating a new buffer block
		 */
		if used < usedBefore && used < used0 {
			nextEntry := cur.List.Next
			nextInterface := ContainerOf(nextEntry, &BufferBlock{}, "List")
			cur, _ = nextInterface.(*BufferBlock)
			continue
		}

		if usedmax < used {
			bb = cur
			usedmax = used
		}

		nextEntry := cur.List.Next
		nextInterface := ContainerOf(nextEntry, &BufferBlock{}, "List")
		cur, _ = nextInterface.(*BufferBlock)
	}

	return bb, nil
}

// BhBalloon expands a buffer head
func BhBalloon(bh *BufferHead, incr uint64) error {
	if bh == nil {
		return fmt.Errorf("nil buffer head")
	}

	block := bh.Block
	if bh.Off == block.Buffers.Off {
		block.Buffers.Off += incr
	}
	return nil
}

// MapBh maps a buffer block
func MapBh(bmgr *BufferManager, bb *BufferBlock) uint32 {
	if bmgr == nil {
		// When called with nil bmgr, we just want to ensure bb is assigned to block 0
		if bb != nil {
			bb.BlkAddr = 0 // Assign to block 0 for superblock
		}
		return 0
	}

	if bb == nil {
		return bmgr.TailBlkAddr
	}

	if bb.BlkAddr != NULL_ADDR {
		return bb.BlkAddr
	}

	var blkAddr uint32
	blkSize := bmgr.Sbi.ErofsBlockSize()

	if bb.Type == META {
		blkAddr = bmgr.TailBlkAddr
		bmgr.TailBlkAddr++
		bmgr.MetaBlkCnt++

	} else {
		// Walk backward to reuse free block slots
		bucketIndex := blkSize % uint64(len(bmgr.MappedBuckets[0]))
		head := bmgr.MappedBuckets[bb.Type][bucketIndex].Prev

		if head != &bmgr.MappedBuckets[bb.Type][bucketIndex] {
			tInterface := ContainerOf1(head, &BufferBlock{}, "MappedList")
			t, ok := tInterface.(*BufferBlock)
			if !ok {
				// Handle error - fall back to default behavior
				blkAddr = bmgr.TailBlkAddr
				bmgr.TailBlkAddr++
			} else {
				blkAddr = t.BlkAddr + 1
			}
		} else {
			blkAddr = bmgr.TailBlkAddr
			bmgr.TailBlkAddr++
		}
	}

	bb.BlkAddr = blkAddr

	// Add to mapped bucket
	bucketIndex := blkSize % uint64(len(bmgr.MappedBuckets[0]))
	ListAdd(&bb.MappedList, &bmgr.MappedBuckets[bb.Type][bucketIndex])
	bmgr.LastMappedBlock = bb

	return blkAddr
}

// BhTell returns the offset of a buffer head
func BhTell(bh *BufferHead, end bool) uint64 {
	bb := bh.Block
	bmgr := bb.Buffers.FsPrivate.(*BufferManager)

	if bb.BlkAddr == 0xFFFFFFFF { // NULL_ADDR
		return 0xFFFFFFFFFFFFFFFF // NULL_ADDR_UL
	}

	pos := uint64(bb.BlkAddr) << bmgr.Sbi.BlkSzBits
	if end {
		// Get next entry's offset
		// This is a placeholder - you'll need to implement list navigation
		return pos + 0 // Offset of next entry
	}
	return pos + bh.Off
}

// BDrop drops a buffer head
func BDrop(bh *BufferHead, tryRevoke bool) {
	if bh == nil {
		return
	}

	bb := bh.Block

	// Call flush operation if present
	if bh.Op != nil && bh.Op.Flush != nil {
		ret := bh.Op.Flush(bh)
		if ret < 0 {
			return
		}
	}

	if tryRevoke && bh.Off == bb.Buffers.Off {
		// Check if the bh can be revoked - must be the last one
		if ListIsLast(&bh.List, &bb.Buffers.List) {
			bb.Buffers.Off = bh.Off
			ListDel(&bh.List)

			// Check if the buffer block is still in use
			if IsListEmpty(&bb.Buffers.List) {
				ListDel(&bb.List)

				// Remove from mapped list if needed
				if bb.BlkAddr != NULL_ADDR {
					ListDel(&bb.MappedList)
				}

				// In Go, we rely on garbage collection instead of free()
				bb = nil
			}

			// In Go, we rely on garbage collection
			return
		}
	}

	ListDel(&bh.List)
	// Let Go's garbage collector handle the memory
}

// ContainerOf is a Go implementation of the C container_of macro
// It returns a pointer to the struct that contains the given member
// ptr: pointer to the member
// sample: a zero value of the container type
// member: the name of the member within the struct
func ContainerOf1(ptr, sample interface{}, member string) interface{} {
	// Get the type of the sample
	sampleValue := reflect.ValueOf(sample)
	if sampleValue.Kind() == reflect.Ptr {
		sampleValue = sampleValue.Elem()
	}

	// Find the field by name
	field := sampleValue.FieldByName(member)
	if !field.IsValid() {
		return nil
	}

	// Get the offset of the field within the struct
	fieldOffset := field.UnsafeAddr() - sampleValue.UnsafeAddr()

	// Get the address of the member pointer
	ptrValue := reflect.ValueOf(ptr)
	memberAddr := ptrValue.Pointer()

	// Calculate the address of the container struct
	containerAddr := memberAddr - uintptr(fieldOffset)

	// Create a new pointer to the container type
	containerType := reflect.PointerTo(sampleValue.Type())
	containerPtr := reflect.NewAt(containerType.Elem(), unsafe.Pointer(containerAddr))

	// Return the container pointer
	return containerPtr.Interface()
}

func RoundDown(x, y int) int {
	return x - (x % y)
}
