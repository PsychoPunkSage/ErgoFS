package types

import (
	"time"
)

// Inode represents an EroFS inode
type Inode struct {
	Sbi    *SuperBlkInfo
	Parent *Inode

	// Basic inode attributes
	Mode      uint16
	Size      uint64
	Ino       [2]uint64
	Uid       uint32
	Gid       uint32
	Mtime     uint64
	MtimeNsec uint32
	Nlink     uint32

	// Union for block address or device ID
	BlkAddr uint32
	Blocks  uint32
	Rdev    uint32

	// Data layout
	DataLayout      uint8
	InodeIsize      uint8
	IdataSize       uint16
	DataSource      uint8
	CompressedIdata bool
	LazyTailblock   bool
	Opaque          bool
	Whiteouts       bool

	// Extended attributes
	XattrIsize  uint32
	ExtentIsize uint32

	// NID and buffer heads
	Nid      uint64
	Bh       *BufferHead
	BhInline *BufferHead
	BhData   *BufferHead

	// Compression info
	ZAdvise              uint16
	ZAlgorithmType       [2]uint8
	ZLogicalClusterBits  uint8
	ZPhysicalClusterBlks uint8
}

// NewInode creates a new inode with default values
func NewInode(sbi *SuperBlkInfo) *Inode {
	return &Inode{
		Sbi:        sbi,
		DataLayout: EROFS_INODE_FLAT_PLAIN,
		Mtime:      uint64(time.Now().Unix()),
		MtimeNsec:  uint32(time.Now().Nanosecond()),
	}
}

// IsDir returns true if the inode is a directory
func (i *Inode) IsDir() bool {
	return (i.Mode & 0170000) == 040000 // S_IFDIR
}

// IsReg returns true if the inode is a regular file
func (i *Inode) IsReg() bool {
	return (i.Mode & 0170000) == 0100000 // S_IFREG
}

// IsLnk returns true if the inode is a symbolic link
func (i *Inode) IsLnk() bool {
	return (i.Mode & 0170000) == 0120000 // S_IFLNK
}

// IsCompressed returns true if the inode data is compressed
func (i *Inode) IsCompressed() bool {
	return i.DataLayout == EROFS_INODE_COMPRESSED_FULL ||
		i.DataLayout == EROFS_INODE_COMPRESSED_COMPACT
}

// SetRoot marks the inode as root directory
func (i *Inode) SetRoot() {
	i.Mode = 040755 // directory with 0755 permissions
	i.Nlink = 2     // . and ..
	i.Parent = i    // Root is its own parent
}
