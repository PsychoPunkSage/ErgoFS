package types

import (
	"bytes"
	"fmt"
	"syscall"
	"unsafe"

	errs "github.com/PsychoPunkSage/ErgoFS/pkg/errors"
)

type ErofsFragmentDedupeItem struct {
	list   ListHead
	length uint32
	pos    int64  // erofs_off_t
	data   []byte // Flexible array member in C, slice in Go
}

func ZErofsFragmentsDedupe(inode *ErofsInode, fd int, tofcrc *uint32) int {
	dataToHash := make([]byte, EROFS_TOF_HASHLEN)
	var ret int

	if inode.ISize <= EROFS_TOF_HASHLEN {
		return 0
	}

	offset := int64(inode.ISize - EROFS_TOF_HASHLEN)
	_, err := syscall.Seek(fd, offset, SEEK_SET)
	if err != nil {
		return -int(err.(syscall.Errno))
	}

	ret, err = syscall.Read(fd, dataToHash)
	if ret != EROFS_TOF_HASHLEN {
		return -int(syscall.Errno(err.(syscall.Errno)))
	}

	*tofcrc = ErofsGetCrc32c(^uint32(0), dataToHash, EROFS_TOF_HASHLEN)
	ret = ZErofsFragmentsDedupeFind(inode, fd, *tofcrc)
	if ret < 0 {
		return ret
	}

	_, err = syscall.Seek(fd, 0, SEEK_SET)
	if err != nil {
		return -int(err.(syscall.Errno))
	}
	return 0
}

// zErofsFragmentsDedupeFind is the Go equivalent of z_erofs_fragments_dedupe_find
func ZErofsFragmentsDedupeFind(inode *ErofsInode, fd int, crc uint32) int {
	epi := inode.Sbi.PackedInode
	var di *ErofsFragmentDedupeItem = nil

	// head := &(epi.Hash[FRAGMENT_HASH(uint(crc))])
	hashIndex := FRAGMENT_HASH(uint(crc))
	// Get pointer to the hashIndex-th element
	headPtr := unsafe.Pointer(uintptr(unsafe.Pointer(epi.Hash)) + uintptr(hashIndex)*unsafe.Sizeof(ListHead{}))
	head := (*ListHead)(headPtr)

	var s1 uint64
	var e1 uint32
	var deduped int64
	var data []byte
	var ret int

	if IsListEmpty(head) {
		return 0
	}

	if int64(inode.ISize) < EROFS_FRAGMENT_INMEM_SZ_MAX {
		s1 = uint64(int64(inode.ISize))
	} else {
		s1 = EROFS_FRAGMENT_INMEM_SZ_MAX
	}

	data = make([]byte, s1)
	if data == nil {
		return -errs.ENOMEM
	}

	ret, err := syscall.Pread(fd, data, int64(inode.ISize)-int64(s1))
	if ret != int(s1) {
		return -int(err.(syscall.Errno))
	}

	e1 = uint32(s1) - EROFS_TOF_HASHLEN
	deduped = 0

	// Iterate through the linked list
	for ptr := head.Next; ptr != head; ptr = ptr.Next {
		cur := ListEntry(ptr, ErofsFragmentDedupeItem{}, "list").(*ErofsFragmentDedupeItem)

		var e2, mn uint32
		var i, pos int64

		if cur.length <= EROFS_TOF_HASHLEN {
			// dbgBugOn(true) // Equivalent to DBG_BUGON
			continue
		}

		e2 = cur.length - EROFS_TOF_HASHLEN

		if !bytes.Equal(data[e1:e1+EROFS_TOF_HASHLEN], cur.data[e2:e2+EROFS_TOF_HASHLEN]) {
			continue
		}

		i = 0
		if e1 < e2 {
			mn = e1
		} else {
			mn = e2
		}

		for i < int64(mn) && cur.data[int64(e2)-i-1] == data[int64(e1)-i-1] {
			i++
		}

		i += int64(EROFS_TOF_HASHLEN)
		if i >= int64(s1) { // full short match
			// dbgBugOn(i > int64(s1))
			pos = cur.pos + int64(cur.length) - int64(s1)

			for i < int64(inode.ISize) && pos > 0 {
				buf := [2][16384]byte{}
				var sz uint64

				if uint64(pos) < uint64(len(buf[0])) {
					sz = uint64(pos)
				} else {
					sz = uint64(len(buf[0]))
				}

				if uint64(int64(inode.ISize)-i) < sz {
					sz = uint64(int64(inode.ISize) - i)
				}

				n, err := syscall.Pread(epi.Fd, buf[0][:sz], pos-int64(sz))
				if n != int(sz) || err != nil {
					break
				}

				n, err = syscall.Pread(fd, buf[1][:sz], int64(inode.ISize)-i-int64(sz))
				if n != int(sz) || err != nil {
					break
				}

				if !bytes.Equal(buf[0][:sz], buf[1][:sz]) {
					break
				}

				pos -= int64(sz)
				i += int64(sz)
			}
		}

		if i <= deduped {
			continue
		}

		di = cur
		deduped = i
		if deduped == int64(inode.ISize) {
			break
		}
	}

	// In Go, we don't need to free data as it will be garbage collected

	if deduped > 0 {
		// dbgBugOn(di == nil)
		inode.FragmentSize = deduped
		inode.Fragmentoff = di.pos + int64(di.length) - deduped
		// erofsDbg("Dedupe %d tail data at %d", inode.FragmentSize, inode.Fragmentoff)
		fmt.Printf("Dedupe %d tail data at %d\n", inode.FragmentSize, inode.Fragmentoff)
	}

	return 0
}
