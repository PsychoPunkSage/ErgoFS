package types

import (
	"fmt"
	"os"
	"sync"
	"syscall"
	"time"
	"unsafe"
)

type ErofsDiskbuf struct {
	Sp     unsafe.Pointer // Internal stream pointer
	Offset uint64         // Internal offset
}

// Inode represents an EroFS inode
type ErofsInode struct {
	// Linked lists for hash, subdirectories, and extended attributes
	IHash    ListHead
	ISubdirs ListHead
	IXattrs  ListHead

	// Union in C is represented as a struct with all fields
	// Runtime flags or next pointer for directory dumping
	Flags        uint32
	NextDirWrite *ErofsInode

	// Atomic counter for reference counting
	ICount int32 // Using Go's atomic package for operations

	// File system and parent pointers
	Sbi     *SuperBlkInfo
	IParent *ErofsInode

	// Device ID containing source file (for mkfs.erofs)
	Dev uint32

	// Standard inode attributes
	IMode      uint16    // Mode and permissions
	ISize      uint64    // File size
	IIno       [2]uint64 // Inode number (array of 2 uint64)
	IUid       uint32    // User ID
	IGid       uint32    // Group ID
	IMtime     uint64    // Modification time
	IMtimeNsec uint32    // Nanosecond part of modification time
	INlink     uint32    // Number of hard links

	// Union in C, represented as individual fields in Go
	IBlkaddr uint32 // Block address
	IBlocks  uint32 // Number of blocks
	IRdev    uint32 // Device ID for special files

	// Chunk information
	ChunkFormat uint16
	ChunkBits   uint8

	// Paths and links
	ISrcpath string        // Source path
	ILink    string        // Symbolic link target
	IDiskbuf *ErofsDiskbuf // Disk buffer

	// Data layout and size information
	DataLayout      uint8
	InodeIsize      uint8
	IdataSize       uint16
	DataSource      uint8 // PPS:: No idea what to store (char -> rune/uint8/int8)
	CompressedIdata bool
	LazyTailblock   bool
	Opaque          bool
	Whiteouts       bool

	// Extended attributes
	XattrIsize  uint32
	ExtentIsize uint32

	XattrSharedCount  uint32
	XattrSharedXattrs *uint32

	// NID and buffer heads
	Nid      uint64
	Bh       *BufferHead
	BhInline *BufferHead
	BhData   *BufferHead

	// Inline data
	Idata unsafe.Pointer

	// EOF tail packing data
	EofTailraw     unsafe.Pointer
	EofTailrawsize uint32

	// Chunk indexes and compression metadata
	ChunkIndexes unsafe.Pointer

	// Compression fields
	ZAdvise              uint16
	ZAlgorithmType       [2]uint8
	ZLogicalClusterbits  uint8
	ZPhysicalClusterblks uint8
	ZTailextentHeadlcn   uint64
	FragmentSize         int64 // Using same type as erofs_off_t
	ZIdataoff            uint32
	Fragmentoff          int64 // Using same type as erofs_off_t
	// z_idata_size is mapped to IdataSize as mentioned in the C macro

	Compressmeta unsafe.Pointer

	// Android-specific capabilities
	// #ifdef WITH_ANDROID is represented as a regular field
	Capabilities uint64
}

type ErofsInodeExtended struct {
	IFormat      uint16 // inode format hints
	IXattrICount uint16 // inline xattr count
	IMode        uint16
	IReserved    uint16
	ISize        uint64
	IU           ErofsInodeIU // Union in C

	IIno       uint32 // Used for 32-bit stat compatibility
	IUid       uint32
	IGid       uint32
	IMTime     uint64
	IMTimeNsec uint32
	INlink     uint32
	IReserved2 [16]byte // Reserved bytes
}

// ErofsInodeChunkInfo represents chunk block bits and reserved field
type ErofsInodeChunkInfo struct {
	Format   uint16 // chunk blkbits, etc.
	Reserved uint16
}

// ErofsInodeIU represents the union erofs_inode_i_u
type ErofsInodeIU struct {
	CompressedBlocks uint32
	RawBlkAddr       uint32
	Rdev             uint32
	ChunkInfo        ErofsInodeChunkInfo
}

// ErofsInodeCompact represents the 32-byte reduced form of an on-disk inode
type ErofsInodeCompact struct {
	IFormat      uint16       // inode format hints
	IXattrIcount uint16       // Inline xattr count
	IMode        uint16       // Mode
	INlink       uint16       // Number of links
	ISize        uint32       // File size
	IReserved    uint32       // Reserved
	IU           ErofsInodeIU // Union field
	IIno         uint32       // Inode number for 32-bit stat compatibility
	IUid         uint16       // User ID
	IGid         uint16       // Group ID
	IReserved2   uint32       // Additional reserved field
}

type ErofsPackedInode struct {
	// Hash         *ListHead  // hash list
	Hash         *ListHead  // hash list
	Fd           int        // file descriptor
	UptoDate     *uint64    // likely represents a bitmap or array indicating which parts of the inode's data are up-to-date.
	Mutex        sync.Mutex // mutex
	UptodateSize uint32     // represent the size of the uptodate bitmap or array. It indicates how many entries are in the uptodate structure
}

// NewInode creates a new inode with default values
func NewInode(sbi *SuperBlkInfo) *ErofsInode {
	return &ErofsInode{
		Sbi:        sbi,
		DataLayout: EROFS_INODE_FLAT_PLAIN,
		IMtime:     uint64(time.Now().Unix()),
		IMtimeNsec: uint32(time.Now().Nanosecond()),
	}
}

// IsDir returns true if the inode is a directory
func (i *ErofsInode) IsDir() bool {
	return (i.IMode & 0170000) == 040000 // S_IFDIR
}

// IsReg returns true if the inode is a regular file
func (i *ErofsInode) IsReg() bool {
	return (i.IMode & 0170000) == 0100000 // S_IFREG
}

// IsLnk returns true if the inode is a symbolic link
func (i *ErofsInode) IsLnk() bool {
	return (i.IMode & 0170000) == 0120000 // S_IFLNK
}

// IsCompressed returns true if the inode data is compressed
func (i *ErofsInode) IsCompressed() bool {
	return i.DataLayout == EROFS_INODE_COMPRESSED_FULL ||
		i.DataLayout == EROFS_INODE_COMPRESSED_COMPACT
}

// SetRoot marks the inode as root directory
func (i *ErofsInode) SetRoot() {
	i.IMode = 040755 // directory with 0755 permissions
	i.INlink = 2     // . and ..
	i.IParent = i    // Root is its own parent
}

// ErofsReadInodeFromDisk reads an inode from disk and fills the in-memory inode structure
func ErofsReadInodeFromDisk(vi *ErofsInode) error {
	// var ret int64
	// var ifmt uint16

	// DBG_BUGON(vi.Sbi == nil)
	// inodeLoc := erofsIloc(vi)

	// // Create buffer for reading inode data
	// buf := make([]byte, binary.Size(ErofsInodeExtended{}))

	// // Read the compact inode first (which is always the first part)
	// ret, _ = ErofsDevRead(vi.Sbi, 0, buf, inodeLoc, int64(binary.Size(ErofsInodeCompact{})))
	// if ret < 0 {
	// 	return fmt.Errorf("failed to read compact inode: %w", syscall.Errno(-ret))
	// }

	// // Parse compact inode format
	// var dic *ErofsInodeCompact = (*ErofsInodeCompact)(unsafe.Pointer(&buf[0]))
	// // In Go, we need to manually decode the binary data
	// // This is a simplified version - in practice, you'd use binary.Read or a struct decoder
	// ifmt = dic.IFormat

	// Set DataLayout
	// vi.DataLayout = ErofsInodeDatalayout(ifmt)
	// if vi.DataLayout >= EROFS_INODE_DATALAYOUT_MAX {
	// 	return fmt.Errorf("unsupported DataLayout %d of nid %d: %w",
	// 		vi.DataLayout, vi.Nid, syscall.Errno(EOPNOTSUPP))
	// }

	// // Process based on inode version
	// switch ErofsInodeVersion(ifmt) {
	// case EROFS_INODE_LAYOUT_EXTENDED:
	// 	vi.InodeIsize = uint8(binary.Size(ErofsInodeExtended{}))

	// 	// Read the rest of the extended inode
	// 	ret, _ = ErofsDevRead(vi.Sbi, 0, buf[binary.Size(ErofsInodeCompact{}):],
	// 		inodeLoc+uint64(binary.Size(ErofsInodeCompact{})),
	// 		int64(binary.Size(ErofsInodeExtended{})-binary.Size(ErofsInodeCompact{})))
	// 	if ret < 0 {
	// 		return fmt.Errorf("failed to read extended inode: %w", syscall.Errno(-ret))
	// 	}

	// 	// Parse extended inode data
	// 	var die *ErofsInodeExtended = (*ErofsInodeExtended)(unsafe.Pointer(&buf[0]))
	// 	// In practice, you'd use binary.Read for proper decoding
	// 	// This is a simplified version that assumes the buffer contains valid data

	// 	// Extract fields from extended inode (die)
	// 	vi.XattrIsize = ErofsXattrIbodySize(binary.LittleEndian.Uint16(buf[2:4])) // i_xattr_icount
	// 	vi.IMode = die.IMode                                                      // i_mode
	// 	vi.IIno[0] = uint64(die.IIno)                                             // i_ino

	// 	// Handle different file types
	// 	switch vi.IMode & S_IFMT {
	// 	case S_IFREG, S_IFDIR, S_IFLNK:
	// 		vi.IBlkaddr = binary.LittleEndian.Uint32(buf[32:36]) // raw_blkaddr
	// 	case S_IFCHR, S_IFBLK:
	// 		vi.IRdev = ErofsNewDecodeDev(binary.LittleEndian.Uint32(buf[36:40])) // rdev
	// 	case S_IFIFO, S_IFSOCK:
	// 		vi.IRdev = 0
	// 	default:
	// 		return fmt.Errorf("bogus i_mode (%o) @ nid %d: %w", vi.IMode, vi.Nid, syscall.Errno(EFSCORRUPTED))
	// 	}

	// 	// Fill other fields from extended inode
	// 	vi.IUid = die.IUid             // i_uid
	// 	vi.IGid = die.IGid             // i_gid
	// 	vi.INlink = die.INlink         // i_nlink
	// 	vi.IMtime = die.IMTime         // i_mtime
	// 	vi.IMtimeNsec = die.IMTimeNsec // i_mtime_nsec
	// 	vi.ISize = die.ISize           // i_size

	// 	// Fill chunk format for chunk-based inodes
	// 	if vi.DataLayout == EROFS_INODE_CHUNK_BASED {
	// 		vi.ChunkFormat = die.IU.ChunkInfo.Format // c.format
	// 	}

	// case EROFS_INODE_LAYOUT_COMPACT:
	// 	vi.InodeIsize = uint8(binary.Size(ErofsInodeCompact{}))

	// 	// Parse compact inode fields
	// 	// In practice, you'd use binary.Read for proper decoding
	// 	vi.XattrIsize = ErofsXattrIbodySize(dic.IXattrIcount) // i_xattr_icount
	// 	vi.IMode = dic.IMode                                  // i_mode
	// 	vi.IIno[0] = uint64(dic.IIno)                         // i_ino

	// 	// Handle different file types
	// 	switch vi.IMode & S_IFMT {
	// 	case S_IFREG, S_IFDIR, S_IFLNK:
	// 		vi.IBlkaddr = dic.IU.RawBlkAddr // raw_blkaddr
	// 	case S_IFCHR, S_IFBLK:
	// 		vi.IRdev = ErofsNewDecodeDev(dic.IU.Rdev) // rdev
	// 	case S_IFIFO, S_IFSOCK:
	// 		vi.IRdev = 0
	// 	default:
	// 		return fmt.Errorf("bogus i_mode (%o) @ nid %d: %w", vi.IMode, vi.Nid, syscall.Errno(EFSCORRUPTED))
	// 	}

	// 	// Fill other fields from compact inode
	// 	vi.IUid = uint32(dic.IUid)     // i_uid
	// 	vi.IGid = uint32(dic.IGid)     // i_gid
	// 	vi.INlink = uint32(dic.INlink) // i_nlink

	// 	// Use superblock build time for compact inodes
	// 	vi.IMtime = vi.Sbi.BuildTime
	// 	vi.IMtimeNsec = vi.Sbi.BuildTimeNsec

	// 	vi.ISize = uint64(dic.ISize) // i_size

	// 	// Fill chunk format for chunk-based inodes
	// 	if vi.DataLayout == EROFS_INODE_CHUNK_BASED {
	// 		vi.ChunkFormat = dic.IU.ChunkInfo.Format // c.format
	// 	}

	// default:
	// 	return fmt.Errorf("unsupported on-disk inode version %d of nid %d: %w",
	// 		ErofsInodeVersion(ifmt), vi.Nid, syscall.Errno(EOPNOTSUPP))
	// }

	// // Set flags and handle chunk-based inodes
	// vi.Flags = 0
	// if vi.DataLayout == EROFS_INODE_CHUNK_BASED {
	// 	if uint32(vi.ChunkFormat)&^EROFS_CHUNK_FORMAT_ALL != 0 {
	// 		return fmt.Errorf("unsupported chunk format %x of nid %d: %w",
	// 			vi.ChunkFormat, vi.Nid, syscall.Errno(EOPNOTSUPP))
	// 	}
	// 	vi.ChunkBits = uint8(uint32(vi.Sbi.BlkSzBits) + (uint32(vi.ChunkFormat) & EROFS_CHUNK_FORMAT_BLKBITS_MASK))
	// }

	return nil
}

// InitPackedFile initializes packed file handling for the filesystem
func InitPackedFile(sbi *SuperBlkInfo, fragmentsMkfs bool) error {
	// Check if PackedInode is already initialized
	if sbi.PackedInode != nil {
		return fmt.Errorf("packed inode already initialized")
	}

	// Create a new PackedInode
	epi := &ErofsPackedInode{}

	// Store in the superblock info
	sbi.PackedInode = epi

	// Initialize hash table for fragments if needed
	if fragmentsMkfs {
		listHeads := make([]ListHead, FRAGMENT_HASHSIZE)
		epi.Hash = &listHeads[0] // PPS:: Big issue

		// Initialize each list head
		for i := 0; i < FRAGMENT_HASHSIZE; i++ {
			InitListHead(epi.Hash)
		}
	}

	// Create a temporary file
	tmpFile, err := ErofsTempfile()
	if err != nil {
		// Clean up on error
		// ExitPackedFile(sbi)
		return fmt.Errorf("failed to create temporary file: %v", err)
	}
	epi.Fd = tmpFile

	// // Handle existing fragments if needed
	// if HasFeature(sbi, "fragments") && sbi.PackedNid > 0 {
	// 	// Create a temporary inode to read from disk
	// 	ei := &ErofsInode{
	// 		Sbi: sbi,
	// 		Nid: sbi.PackedNid,
	// 	}

	// 	// Read the inode from disk
	// 	err = ReadInodeFromDisk(ei)
	// 	if err != nil {
	// 		Debug(EROFS_ERR, "failed to read packed inode from disk: %v", err)
	// 		// ExitPackedFile(sbi)
	// 		return err
	// 	}

	// 	// Seek to the end of existing data
	// 	offset, err := epi.Fd.Seek(ei.Size, os.SEEK_SET)
	// 	if err != nil {
	// 		// ExitPackedFile(sbi)
	// 		return fmt.Errorf("failed to seek in packed file: %v", err)
	// 	}

	// 	if offset < 0 {
	// 		// ExitPackedFile(sbi)
	// 		return fmt.Errorf("invalid offset in packed file")
	// 	}

	// 	// Calculate uptodate bitmap size and allocate
	// 	epi.UptodateSize = BlockRoundUp(sbi, ei.Size) / 8
	// 	epi.UptoDate = make([]byte, epi.UptoDateSize)
	// }

	return nil
}

func ErofsTempfile() (int, error) {
	// Get the temp dir
	tmpDir := os.Getenv("TMPDIR")
	if tmpDir == "" {
		tmpDir = "/tmp"
	}

	// template for temp file
	// template := filepath.Join(tmpDir, "tmp.XXXXXXXXXX")

	// Create a temporary file with a random name
	// Note: Go's ioutil.TempFile creates a file with a random name
	// that satisfies the pattern
	tmpFile, err := os.CreateTemp(tmpDir, "tmp.*")
	if err != nil {
		return -1, fmt.Errorf("failed to create temp file: %w", err)
	}

	// Get the file descriptor - in Go we work with the os.File struct
	// but we can extract the Unix file descriptor
	fd := int(tmpFile.Fd())

	// Remove the file name from the filesystem
	// The file remains accessible via the file descriptor
	if err := os.Remove(tmpFile.Name()); err != nil {
		tmpFile.Close()
		return -1, fmt.Errorf("failed to unlink temp file: %w", err)
	}

	// Get current umask
	// Note: syscall.Umask is not directly portable to non-Unix systems
	oldUmask := syscall.Umask(0)
	syscall.Umask(oldUmask) // Restore the original umask

	// Change file mode according to 0666 & ~umask
	mode := os.FileMode(0666 &^ os.FileMode(oldUmask))
	if err := tmpFile.Chmod(mode); err != nil {
		tmpFile.Close()
		return -1, fmt.Errorf("failed to chmod temp file: %w", err)
	}

	// We don't close the file since we're returning the fd
	// The caller is responsible for closing it

	return fd, nil
}

// DBG_BUGON is a debug assertion helper
func DBG_BUGON(condition bool) {
	if condition {
		panic("BUG condition detected")
	}
}

// Equivalent to `#define erofs_pos(sbi, nr) ((erofs_off_t)(nr) << (sbi)->blkszbits)`
func erofsPos(sbi *SuperBlkInfo, nr uint64) uint64 {
	return nr << uint64(sbi.BlkSzBits)
}

// Equivalent to `static inline erofs_off_t erofs_iloc(struct erofs_inode *inode)`
func erofsIloc(inode *ErofsInode) uint64 {
	sbi := inode.Sbi
	return erofsPos(sbi, uint64(sbi.MetaBlkAddr)) + (inode.Nid << uint64(sbi.ISlotBits))
}
