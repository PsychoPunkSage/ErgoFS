package util

import (
	"errors"
	"fmt"
	"reflect"
	"unsafe"

	"github.com/PsychoPunkSage/ErgoFS/pkg/types"
)

type ErofsDeviceSlot struct {
	Tag           [64]byte
	Blocks        uint32
	MappedBlkAddr uint32
	Reserved      [56]byte
}

const EROFS_DEVT_SLOT_SIZE = 64 + 4 + 4 + 56

// BufferManagerOps defines buffer manager operations
type BufferManagerOps struct {
	// Add operation callbacks as needed
}

// SkipWriteOps defines operations that skip writing
var SkipWriteOps = &types.BufferHeadOps{
	Flush: func(bh *types.BufferHead) int {
		// Implementation to skip writing
		return 0
	},
}

// InitBufferManager initializes a new buffer manager
func InitBufferManager(sbi *types.SuperBlkInfo, startBlock uint32) *types.BufferManager {
	bmgr := &types.BufferManager{
		Sbi:           sbi,
		TailBlkAddr:   startBlock,
		MappedBuckets: make([][][]types.ListHead, types.META+1),
	}

	// Initialize list heads
	InitListHead(&bmgr.BlkH.List)
	bmgr.BlkH.BlkAddr = 0xFFFFFFFF // NULL_ADDR
	bmgr.LastMappedBlock = &bmgr.BlkH

	// Initialize mapped buckets
	for i := 0; i <= types.META; i++ {
		bmgr.MappedBuckets[i] = make([][]types.ListHead, types.EROFS_MAX_BLOCK_SIZE)
		for j := 0; j < int(types.EROFS_MAX_BLOCK_SIZE); j++ {
			bmgr.MappedBuckets[i][j] = make([]types.ListHead, 0)
			// Initialize each list head
			// This is a placeholder - you'll need to implement proper list initialization
		}
	}

	return bmgr
}

// ReserveSuperblock reserves space for the superblock
func ReserveSuperblock(bmgr *types.BufferManager) (*types.BufferHead, error) {
	bh, err := Balloc(bmgr, types.META, 0, 0, 0)
	if err != nil {
		return nil, fmt.Errorf("failed to allocate super: %v", err)
	}

	bh.Op = SkipWriteOps
	err = BhBalloon(bh, uint64(types.EROFS_SUPER_END))
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
func GetAlignSize(sbi *types.SuperBlkInfo, bufType int) (int, int) {
	if bufType == types.DATA {
		return int(sbi.ErofsBlockSize()), bufType
	}

	if bufType == types.INODE {
		return 32, // Size of struct erofs_inode_compact <PPS::> Need to code it>
			types.META
	} else if bufType == types.DIRA {
		return int(sbi.ErofsBlockSize()), types.META
	} else if bufType == types.XATTR {
		return 4, // Size of struct erofs_xattr_entry <PPS:: need to implement it>
			types.META
	} else if bufType == types.DEVT {
		return int(EROFS_DEVT_SLOT_SIZE), types.META
	}

	if bufType == types.META {
		return 1, bufType
	}

	return -1, 0 // Error
}

// Helper function to allocate a buffer block
func BlkAllocBuf(bmgr *types.BufferManager, aType int) (*types.BufferBlock, error) {
	bb := &types.BufferBlock{
		BlkAddr: types.NULL_ADDR,
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
func Balloc(bmgr *types.BufferManager, bufType int, size uint64, requiredExt, inlineExt uint) (*types.BufferHead, error) {
	var bb *types.BufferBlock
	var bh *types.BufferHead

	alignSize, aType := GetAlignSize(bmgr.Sbi, bufType)
	if alignSize < 0 {
		return nil, fmt.Errorf("invalid buffer type: %d", bufType)
	}

	// Total required extensions
	totalRequiredExt := requiredExt + inlineExt

	// Look for an existing buffer block with enough space
	current := bmgr.BlkH.List.Next
	for current != &bmgr.BlkH.List {
		blockInterface := ContainerOf(current, &types.BufferBlock{}, "List")
		block, ok := blockInterface.(*types.BufferBlock)
		if !ok {
			current = current.Next
			continue
		}

		// Skip if type doesn't match
		if block.Type != aType {
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
		bb, err = BlkAllocBuf(bmgr, aType)
		if err != nil {
			return nil, err
		}
	}

	if size > bb.Buffers.Off {
		return nil, fmt.Errorf("empty buffer block, there should exist at least one buffer head in a buffer block")
	}

	// Create a new buffer head
	bh = &types.BufferHead{
		Op:        nil,
		Block:     bb,
		Off:       size,
		FsPrivate: nil,
	}

	// Add to buffer list
	ListAddTail(&bh.List, &bb.Buffers.List)
	return bh, nil
}

// BhBalloon expands a buffer head
func BhBalloon(bh *types.BufferHead, incr uint64) error {
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
func MapBh(bmgr *types.BufferManager, bb *types.BufferBlock) uint32 {
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

	if bb.BlkAddr != types.NULL_ADDR {
		return bb.BlkAddr
	}

	var blkAddr uint32
	blkSize := bmgr.Sbi.ErofsBlockSize()

	if bb.Type == types.META {
		blkAddr = bmgr.TailBlkAddr
		bmgr.TailBlkAddr++
		bmgr.MetaBlkCnt++
	} else {
		// Walk backward to reuse free block slots
		bucketIndex := blkSize % uint32(len(bmgr.MappedBuckets[0]))
		head := bmgr.MappedBuckets[bb.Type][bucketIndex][0].Prev

		if head != &bmgr.MappedBuckets[bb.Type][bucketIndex][0] {
			tInterface := ContainerOf(head, &types.BufferBlock{}, "MappedList")
			t, ok := tInterface.(*types.BufferBlock)
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
	bucketIndex := blkSize % uint32(len(bmgr.MappedBuckets[0]))
	ListAdd(&bb.MappedList, &bmgr.MappedBuckets[bb.Type][bucketIndex][0])
	bmgr.LastMappedBlock = bb

	return blkAddr
}

// BhTell returns the offset of a buffer head
func BhTell(bh *types.BufferHead, end bool) uint64 {
	bb := bh.Block
	bmgr := bb.Buffers.FsPrivate.(*types.BufferManager)

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
func BDrop(bh *types.BufferHead, tryRevoke bool) {
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
				if bb.BlkAddr != types.NULL_ADDR {
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

// Initialize a list head
func InitListHead(list *types.ListHead) {
	list.Next = list
	list.Prev = list
}

// ListAddTail adds an entry to the end of the list
func ListAddTail(newNode, head *types.ListHead) {
	prev := head.Prev
	head.Prev = newNode
	newNode.Next = head
	newNode.Prev = prev
	prev.Next = newNode
}

// ListAdd adds an entry after the specified head
func ListAdd(newNode, head *types.ListHead) {
	next := head.Next
	head.Next = newNode
	newNode.Prev = head
	newNode.Next = next
	next.Prev = newNode
}

// ListDel deletes an entry from the list
func ListDel(entry *types.ListHead) {
	entry.Prev.Next = entry.Next
	entry.Next.Prev = entry.Prev
	entry.Next = nil
	entry.Prev = nil
}

// ListIsLast checks if an entry is the last one
func ListIsLast(list, head *types.ListHead) bool {
	return list.Next == head
}

// IsListEmpty checks if a list is empty
func IsListEmpty(list *types.ListHead) bool {
	return list.Next == list
}

// ContainerOf is a Go implementation of the C container_of macro
// It returns a pointer to the struct that contains the given member
// ptr: pointer to the member
// sample: a zero value of the container type
// member: the name of the member within the struct
func ContainerOf(ptr, sample interface{}, member string) interface{} {
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
