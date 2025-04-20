package types

import (
	"fmt"
	"reflect"
	"strings"
	"sync/atomic"
	"syscall"
	"unsafe"

	errs "github.com/PsychoPunkSage/ErgoFS/pkg/errors"
	"golang.org/x/sys/unix"
)

var FullpathPrefix int

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
		return nil, fmt.Errorf("ENOSPC: %d", errs.ENOSPC)
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

// TO BE REMOVED
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
			next := ListNextEntryBB(bb)
			// Add checks to ensure next is valid
			if next == nil {
				fmt.Println("Next buffer block is nil")
				return -errs.EINVAL
			}
			// fmt.Printf("Next buffer block: %+v\n", next)
			if next.BlkAddr != NULL_ADDR {
				fmt.Println("BattachInternal Failed: Next block address is not NULL_ADDR")
				return -errs.EINVAL
			}
		}

		blkaddr = bb.BlkAddr
		if blkaddr != NULL_ADDR {
			tailupdate = (bmgr.TailBlkAddr == blkaddr+
				uint32(BlkRoundUp(sbi, boff)))

			if oob > 0 && !tailupdate {
				return -errs.EINVAL
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

// ErofsPos converts a block number to byte position
func ErofsPos(sbi *SuperBlkInfo, nr uint64) uint64 {
	return uint64(nr) << sbi.BlkSzBits
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

	return -errs.EINVAL, 0 // Error
}

// Crc32c calculates CRC32C checksum (Castagnoli polynomial)
func Crc32c(crc uint32, data []byte) uint32 {
	const polynomial uint32 = 0x82F63B78

	for _, b := range data {
		crc ^= uint32(b)
		for i := 0; i < 8; i++ {
			if crc&1 != 0 {
				crc = (crc >> 1) ^ polynomial
			} else {
				crc >>= 1
			}
		}
	}

	return crc
}

func ErofsGetCrc32c(crc uint32, in []byte, length int) uint32 {
	for i := 0; i < length; i++ {
		crc ^= uint32(in[i])
		for j := 0; j < 8; j++ {
			if crc&1 != 0 {
				crc = (crc >> 1) ^ CRC32C_POLY_LE
			} else {
				crc = crc >> 1
			}
		}
	}
	return crc
}

func lseek(fd int, offset int64, whence int) int64 {
	// This would typically use the syscall package in Go
	// return syscall.Seek(fd, offset, whence)
	// Or use os.File.Seek if working with os.File objects
	// For compatibility with the original C code signature
	off, err := syscall.Seek(fd, offset, whence)
	if err != nil {
		return -errs.EINVAL
	}
	return int64(off)
}

func ErofsFspath(fullpath string) string {
	// Skip prefix characters
	if FullpathPrefix >= len(fullpath) {
		return ""
	}
	s := fullpath[FullpathPrefix:]

	// Trim leading slashes
	return strings.TrimLeft(s, "/")
}

// Major returns the major device number from a device ID
func Major(dev uint64) uint32 {
	return unix.Major(dev)
}

// Minor returns the minor device number from a device ID
func Minor(dev uint64) uint32 {
	return unix.Minor(dev)
}

// IS_ROOT checks if an inode is the root inode
func IS_ROOT(inode *ErofsInode) bool {
	return inode == erofsParentInode(inode)
}

// erofsParentInode gets the parent inode with the lowest bit masked off
func erofsParentInode(inode *ErofsInode) *ErofsInode {
	// In Go, we need to use uintptr for pointer arithmetic
	ptr := uintptr(unsafe.Pointer(inode.IParent))
	// Clear the lowest bit (equivalent to & ~1UL in C)
	ptr &= ^uintptr(1)
	return (*ErofsInode)(unsafe.Pointer(ptr))
}

func ErofsAtomicDecReturn(InodeICount *int32) uint {
	fmt.Println("ErofsAtomicDecReturn")
	return erofsAtomicSubReturn(InodeICount, 1)
}

func erofsAtomicSubReturn(ptr *int32, i int32) uint {
	fmt.Println("ErofsAtomicDecReturn - Internal")
	return uint(atomic.AddInt32(ptr, ^(i - 1)))
}
