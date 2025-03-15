package types

import (
	"fmt"
	"reflect"
	"unsafe"
)

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

		ret = BattachInternal(cur, nil, size, alignsize, requiredExt+inlineExt, true)
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

		ret = BattachInternal(cur, nil, size, alignsize, requiredExt+inlineExt, true)
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

func ErofsBfree(bb *BufferBlock) {
	bmgr := bb.Buffers.FsPrivate.(*BufferManager)

	if IsListEmpty(&bb.Buffers.List) {
		return
	}

	// If this is the last mapped block, update the pointer to the previous entry
	if bb == bmgr.LastMappedBlock {
		// Get the previous entry in the list
		prev := ListEntry(bb.MappedList.Prev, &BufferBlock{}, "MappedList")
		bmgr.LastMappedBlock = prev.(*BufferBlock)
	}

	// Remove from lists
	ListDel(&bb.MappedList)
	ListDel(&bb.List)

	// In Go, memory is managed by the garbage collector
	// No explicit free is needed, but we can nil the pointer
	// to help the GC and to be explicit about our intentions
	bb = nil
}

func BhFlushGenericEnd(bh *BufferHead) int {
	ListDel(&bh.List)
	bh = nil
	return 0
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

// BupdateMapped updates the mapped buffer block
func BupdateMapped(bb *BufferBlock) {
	bmgr := bb.Buffers.FsPrivate.(*BufferManager)
	sbi := bmgr.Sbi
	var bkt *ListHead

	if bb.BlkAddr == NULL_ADDR {
		return
	}

	bucketIndex := int(bb.Buffers.Off & uint64(ErofsBlkSiz(sbi)-1))
	bkt = &bmgr.MappedBuckets[bb.Type][bucketIndex]

	ListDel(&bb.MappedList)
	ListAddTail(&bb.MappedList, bkt)
}

// BattachInternal attaches a buffer head to a buffer block and returns occupied bytes if successful
func BattachInternal(bb *BufferBlock, bh *BufferHead, incr uint64,
	alignsize, extrasize uint32, dryrun bool) int {
	bmgr := bb.Buffers.FsPrivate.(*BufferManager)
	sbi := bmgr.Sbi
	blksiz := ErofsBlkSiz(sbi)
	blkmask := blksiz - 1
	boff := bb.Buffers.Off
	alignedoffset := RoundUp(boff, uint64(alignsize))

	// Calculate the "out of bounds" condition
	prevBlockOffset := ((boff - 1) & uint64(blkmask)) + 1
	roundedPrevOffset := RoundUp(prevBlockOffset, uint64(alignsize))
	totalSize := roundedPrevOffset + incr + uint64(extrasize)
	oob := Cmpsgn(totalSize, uint64(blksiz))

	var tailupdate bool
	var blkaddr uint32

	if oob >= 0 {
		// The next buffer block should be NULL_ADDR all the time
		if oob > 0 {
			nextEntry := bb.List.Next
			nextInterface := ContainerOf(nextEntry, &BufferBlock{}, "List")
			next, _ := nextInterface.(*BufferBlock)

			if next.BlkAddr != NULL_ADDR {
				return EINVAL
			}
		}

		blkaddr = bb.BlkAddr
		if blkaddr != NULL_ADDR {
			tailupdate = (bmgr.TailBlkAddr == blkaddr+
				uint32(BlkRoundUp(sbi, boff)))

			if oob > 0 && !tailupdate {
				return EINVAL
			}
		}
	}

	if !dryrun {
		if bh != nil {
			bh.Off = alignedoffset
			bh.Block = bb
			ListAddTail(&bh.List, &bb.Buffers.List)
		}
		boff = alignedoffset + incr
		bb.Buffers.Off = boff
		// Need to update the tail_blkaddr
		if tailupdate {
			bmgr.TailBlkAddr = blkaddr + uint32(BlkRoundUp(sbi, boff))
		}
		BupdateMapped(bb)
	}

	return int(((alignedoffset + incr - 1) & uint64(blkmask)) + 1)
}

// Roundup rounds up a number to the nearest multiple of align
func RoundUp(x, y uint64) uint64 {
	if y == 0 {
		return x // Avoid division by zero
	}
	return ((x + (y - 1)) / y) * y
}

func RoundDown(x, y int) int {
	return x - (x % y)
}

// Cmpsgn compares two numbers and returns sign (negative, zero, positive)
func Cmpsgn(a, b uint64) int {
	if a < b {
		return -1
	} else if a > b {
		return 1
	}
	return 0
}

// BlkRoundUp rounds up offset to the next block
func BlkRoundUp(sbi *SuperBlkInfo, offset uint64) uint64 {
	blksz := ErofsBlkSiz(sbi)
	return RoundUp(offset, uint64(blksz)) >> sbi.BlkSzBits
}

// ErofsPos converts a block number to byte position
func ErofsPos(sbi *SuperBlkInfo, nr uint64) uint64 {
	return uint64(nr) << sbi.BlkSzBits
}
