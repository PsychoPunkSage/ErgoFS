package types

import (
	"encoding/binary"
	"time"
)

// InodeCompact represents the on-disk compact inode structure
type InodeCompact struct {
	Format          uint16 // Inode format
	XattrInodeCount uint16 // Inode xattr count
	AccessMode      uint16 // Permissions + file type
	NLink           uint16 // Hard links count
	Uid             uint16 // Owner's user ID
	Gid             uint16 // Owner's group ID
	Reserved        uint32 // Reserved for extension
	ModifyTime      uint64 // Inode modification time
	ModifyNSec      uint32 // Modification time, nanosecond part
	DevSlot         uint32 // Block device slot
}

// InodeExtended represents the on-disk extended inode structure
type InodeExtended struct {
	InodeCompact        // Embedded compact inode
	Size         uint64 // File size
}

// ZIndexTail represents the tail for z-erofs indexing
type ZIndexTail struct {
	BlkAddr  uint32 // Block address for the chunk
	Reserved uint32 // Reserved for extension
}

// XattrHeader represents the header for extended attributes
type XattrHeader struct {
	Magic    uint32 // Magic number for the xattr
	Checksum uint32 // CRC32C checksum
}

// XattrEntry represents an extended attribute entry
type XattrEntry struct {
	NameOff   uint16 // Name offset in the string table
	NameLen   uint8  // Name length
	NameHash  uint8  // Name hash
	ValueSize uint16 // Size of value
	Reserved  uint16 // Reserved for extension
}

// StatInfo represents file stat information
type StatInfo struct {
	Ino       uint64    // Inode number
	Mode      uint16    // File mode
	Type      uint8     // File type (from EROFS_FT_*)
	Uid       uint32    // User ID
	Gid       uint32    // Group ID
	Size      uint64    // File size
	ModTime   time.Time // Modification time
	NLink     uint32    // Number of hard links
	RdevMajor uint32    // Major device number for special files
	RdevMinor uint32    // Minor device number for special files
}

// DirEntry represents a directory entry
type DirEntry struct {
	Nid      uint64 // Target inode number
	NameLen  uint16 // Name length
	FileType uint8  // File type (from EROFS_FT_*)
	Reserved uint8  // Reserved for extension
	Name     string // Entry name
}

// Inode represents an in-memory inode
type Inode struct {
	Nid        uint64            // Inode number
	BlockAddr  uint32            // Block address for the inode
	Stat       StatInfo          // File statistics
	DataLayout uint8             // Data layout (EROFS_INODE_FLAT_*)
	Xattrs     map[string][]byte // Extended attributes
	Data       []byte            // Inode data
	Parent     *Inode            // Parent inode (for non-root inodes)
	Children   []*Inode          // Child inodes (for directories)
	Name       string            // File name (basename)
	SourcePath string            // Source file path
}

// NewInode creates a new inode
func NewInode(nid uint64, name string, stat StatInfo) *Inode {
	return &Inode{
		Nid:        nid,
		Stat:       stat,
		DataLayout: EROFS_INODE_FLAT_PLAIN,
		Xattrs:     make(map[string][]byte),
		Name:       name,
		SourcePath: name,
		Children:   make([]*Inode, 0),
	}
}

// AddChild adds a child inode to a directory inode
func (inode *Inode) AddChild(child *Inode) {
	child.Parent = inode
	inode.Children = append(inode.Children, child)
}

// IsDirectory returns true if the inode is a directory
func (inode *Inode) IsDirectory() bool {
	return inode.Stat.Type == EROFS_FT_DIR
}

// IsRegular returns true if the inode is a regular file
func (inode *Inode) IsRegular() bool {
	return inode.Stat.Type == EROFS_FT_REG_FILE
}

// IsSymlink returns true if the inode is a symlink
func (inode *Inode) IsSymlink() bool {
	return inode.Stat.Type == EROFS_FT_SYMLINK
}

// AddXattr adds an extended attribute
func (inode *Inode) AddXattr(name string, value []byte) {
	inode.Xattrs[name] = value
}

// EncodeCompactInode encodes a compact inode to binary
func EncodeCompactInode(inode *Inode) []byte {
	data := make([]byte, 32) // Size of InodeCompact

	// Format
	binary.LittleEndian.PutUint16(data[0:2], uint16(inode.DataLayout))

	// XattrInodeCount
	binary.LittleEndian.PutUint16(data[2:4], uint16(len(inode.Xattrs)))

	// AccessMode
	binary.LittleEndian.PutUint16(data[4:6], inode.Stat.Mode)

	// NLink
	binary.LittleEndian.PutUint16(data[6:8], uint16(inode.Stat.NLink))

	// Uid and Gid
	binary.LittleEndian.PutUint16(data[8:10], uint16(inode.Stat.Uid))
	binary.LittleEndian.PutUint16(data[10:12], uint16(inode.Stat.Gid))

	// Reserved
	binary.LittleEndian.PutUint32(data[12:16], 0)

	// ModifyTime
	binary.LittleEndian.PutUint64(data[16:24], uint64(inode.Stat.ModTime.Unix()))

	// ModifyNSec
	binary.LittleEndian.PutUint32(data[24:28], uint32(inode.Stat.ModTime.Nanosecond()))

	// DevSlot
	binary.LittleEndian.PutUint32(data[28:32], 0)

	return data
}

// EncodeExtendedInode encodes an extended inode to binary
func EncodeExtendedInode(inode *Inode) []byte {
	data := make([]byte, 40) // Size of InodeExtended

	// First encode the compact part
	compactData := EncodeCompactInode(inode)
	copy(data, compactData)

	// Size
	binary.LittleEndian.PutUint64(data[32:40], inode.Stat.Size)

	return data
}

// EncodeDirEntries encodes directory entries to binary
func EncodeDirEntries(entries []DirEntry) []byte {
	totalSize := 0
	for _, entry := range entries {
		// 12 bytes for fixed header + name length
		entrySize := 12 + len(entry.Name)
		// Align to 4 bytes
		entrySize = (entrySize + 3) & ^3
		totalSize += entrySize
	}

	data := make([]byte, totalSize)
	offset := 0

	for _, entry := range entries {
		// Nid (8 bytes)
		binary.LittleEndian.PutUint64(data[offset:offset+8], entry.Nid)

		// NameLen (2 bytes)
		binary.LittleEndian.PutUint16(data[offset+8:offset+10], entry.NameLen)

		// FileType and Reserved (1 byte each)
		data[offset+10] = entry.FileType
		data[offset+11] = entry.Reserved

		// Name
		nameBytes := []byte(entry.Name)
		copy(data[offset+12:], nameBytes)

		// Move to next entry with alignment
		offset += 12 + len(entry.Name)
		offset = (offset + 3) & ^3
	}

	return data
}
