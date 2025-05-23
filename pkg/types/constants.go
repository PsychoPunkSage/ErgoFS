package types

import (
	"math"
	"unsafe"
)

// EROFS filesystem constants derived from the C implementation
// Many of these constants come from the C header files in the erofs-utils project
/*
Ref file:
	- https://github.com/erofs/erofs-utils/blob/dev/include/erofs_fs.h
	- https://github.com/erofs/erofs-utils/blob/dev/include/erofs/internal.h
	- https://github.com/erofs/erofs-utils/blob/dev/include/erofs/config.h
	- https://github.com/erofs/erofs-utils/blob/dev/include/erofs/cache.h
	- https://github.com/erofs/erofs-utils/blob/dev/include/erofs/dir.h
	- https://github.com/erofs/erofs-utils/blob/dev/include/erofs/fragments.h
	- https://github.com/erofs/erofs-utils/blob/dev/include/erofs/io.h
	- https://github.com/erofs/erofs-utils/blob/dev/include/erofs/print.h
	- https://github.com/erofs/erofs-utils/blob/dev/include/erofs/tar.h

*/

// Block and filesystem constants
const (

	// EROFS_MAX_BLOCK_SIZE is the maximum block size supported by EROFS
	EROFS_MAX_BLOCK_SIZE uint32 = 4096
	EROFS_MIN_BLOCK_SIZE uint32 = 512
	PATH_MAX             uint32 = 4096

	EROFS_ISLOBITS uint32 = 5
	EROFS_SLOTSIZE        = 1 << EROFS_ISLOBITS

	NULL_ADDR    = ^uint32(0)
	NULL_ADDR_UL = ^uint64(0)

	// EROFS_SUPER_END = EROFS_SUPER_OFFSET + uint64(unsafe.Sizeof(erofsSuperBlock{}))

	// // Common block sizes as powers of 2 <defined by me>
	// EROFS_BLKSIZ_BITS_MIN uint8 = 9  // 512 bytes
	// EROFS_BLKSIZ_BITS_DEF uint8 = 12 // 4096 bytes

	// Common block sizes (these are derived from the code logic, not explicit constants)
)

// Superblock feature flags
const (
	// EROFS_SUPER_MAGIC_V1 uint32 = 0xE0F5E1E0
	EROFS_SUPER_MAGIC_V1 uint32 = 0xE0F5E1E2
	EROFS_SUPER_OFFSET   uint32 = 1024
	EROFS_SUPER_END             = EROFS_SUPER_OFFSET + uint32(unsafe.Sizeof(SuperBlock{}))
	// EROFS_SUPER_END = EROFS_SUPER_OFFSET + 128

	EROFS_SB_EXTSLOT_SIZE uint32 = 16

	// Feature compatibility flags
	EROFS_FEATURE_COMPAT_SB_CHKSUM    uint32 = 0x00000001
	EROFS_FEATURE_COMPAT_MTIME        uint32 = 0x00000002
	EROFS_FEATURE_COMPAT_XATTR_FILTER uint32 = 0x00000004

	// Feature incompatibility flags
	EROFS_FEATURE_INCOMPAT_ZERO_PADDING   uint32 = 0x00000001
	EROFS_FEATURE_INCOMPAT_COMPR_CFGS     uint32 = 0x00000002
	EROFS_FEATURE_INCOMPAT_BIG_PCLUSTER   uint32 = 0x00000002
	EROFS_FEATURE_INCOMPAT_CHUNKED_FILE   uint32 = 0x00000004
	EROFS_FEATURE_INCOMPAT_DEVICE_TABLE   uint32 = 0x00000008
	EROFS_FEATURE_INCOMPAT_COMPR_HEAD2    uint32 = 0x00000008
	EROFS_FEATURE_INCOMPAT_ZTAILPACKING   uint32 = 0x00000010
	EROFS_FEATURE_INCOMPAT_FRAGMENTS      uint32 = 0x00000020
	EROFS_FEATURE_INCOMPAT_DEDUPE         uint32 = 0x00000020
	EROFS_FEATURE_INCOMPAT_XATTR_PREFIXES uint32 = 0x00000040
	EROFS_ALL_FEATURE_INCOMPAT            uint32 = EROFS_FEATURE_INCOMPAT_ZERO_PADDING |
		EROFS_FEATURE_INCOMPAT_COMPR_CFGS |
		EROFS_FEATURE_INCOMPAT_BIG_PCLUSTER |
		EROFS_FEATURE_INCOMPAT_CHUNKED_FILE |
		EROFS_FEATURE_INCOMPAT_DEVICE_TABLE |
		EROFS_FEATURE_INCOMPAT_COMPR_HEAD2 |
		EROFS_FEATURE_INCOMPAT_ZTAILPACKING |
		EROFS_FEATURE_INCOMPAT_FRAGMENTS |
		EROFS_FEATURE_INCOMPAT_DEDUPE |
		EROFS_FEATURE_INCOMPAT_XATTR_PREFIXES

	EROFS_I_VERSION_MASK    = 0x01
	EROFS_I_DATALAYOUT_MASK = 0x07

	EROFS_I_VERSION_BIT    = 0
	EROFS_I_DATALAYOUT_BIT = 1
	EROFS_I_ALL_BIT        = 4

	EROFS_I_ALL = (1 << EROFS_I_ALL_BIT) - 1

	EROFS_CHUNK_FORMAT_BLKBITS_MASK uint32 = 0x001F
	EROFS_CHUNK_FORMAT_INDEXES      uint32 = 0x0020
	EROFS_CHUNK_FORMAT_ALL          uint32 = EROFS_CHUNK_FORMAT_BLKBITS_MASK | EROFS_CHUNK_FORMAT_INDEXES

	EROFS_INODE_LAYOUT_COMPACT  = 0
	EROFS_INODE_LAYOUT_EXTENDED = 1
)

// Inode const.
const (
	EROFS_INODE_FLAT_PLAIN         = 0
	EROFS_INODE_COMPRESSED_FULL    = 1
	EROFS_INODE_FLAT_INLINE        = 2
	EROFS_INODE_COMPRESSED_COMPACT = 3
	EROFS_INODE_CHUNK_BASED        = 4
	EROFS_INODE_DATALAYOUT_MAX     = 5
	NR_INODE_HASHTABLE             = 16384
)

// seek
const (
	SEEK_SET  = 0 /* Seek from beginning of file.  */
	SEEK_CUR  = 1 /* Seek from current position.  */
	SEEK_END  = 2 /* Seek from end of file.  */
	SEEK_DATA = 3 /* Seek to next data.  */
	SEEK_HOLE = 4 /* Seek to next hole.  */
)

// Erofs_FT
const (
	EROFS_FT_UNKNOWN = iota
	EROFS_FT_REG_FILE
	EROFS_FT_DIR
	EROFS_FT_CHRDEV
	EROFS_FT_BLKDEV
	EROFS_FT_FIFO
	EROFS_FT_SOCK
	EROFS_FT_SYMLINK
	EROFS_FT_MAX
)

// INODE Data const.
const (
	EROFS_INODE_DATA_SOURCE_NONE      = 0
	EROFS_INODE_DATA_SOURCE_LOCALPATH = 1
	EROFS_INODE_DATA_SOURCE_DISKBUF   = 2
	EROFS_INODE_DATA_SOURCE_RESVSP    = 3
)

// Erofs Advice
const (
	Z_EROFS_ADVISE_COMPACTED_2B        = 0x0001
	Z_EROFS_ADVISE_BIG_PCLUSTER_1      = 0x0002
	Z_EROFS_ADVISE_BIG_PCLUSTER_2      = 0x0004
	Z_EROFS_ADVISE_INLINE_PCLUSTER     = 0x0008
	Z_EROFS_ADVISE_INTERLACED_PCLUSTER = 0x0010
	Z_EROFS_ADVISE_FRAGMENT_PCLUSTER   = 0x0020
)

const (
	CRC32C_POLY_LE = 0x82F63B78
)

const (
	EROFS_I_EA_INITED = 1 << 0
	EROFS_I_Z_INITED  = 1 << 1
)

const (
	EROFS_PACKED_NID_UNALLOCATED = 0
)

// config const.
const (
	// config.h
	EROFS_MAX_COMPR_CFGS uint32 = 64

	FORCE_INODE_COMPACT  = 1
	FORCE_INODE_EXTENDED = 2

	FORCE_INODE_BLOCK_MAP   = 1
	FORCE_INODE_CHUNK_INDEX = 2

	TIMESTAMP_UNSPECIFIED = iota
	TIMESTAMP_NONE        // 1
	TIMESTAMP_FIXED       // 2
	TIMESTAMP_CLAMPING    // 3

	FRAGDEDUPE_FULL  = iota
	FRAGDEDUPE_INODE // 1
	FRAGDEDUPE_OFF   // 2
)

// ReadDIR const.
const (
	// dir.h
	EROFS_READDIR_VALID_PNID   = 0x0001
	EROFS_READDIR_DOTDOT_FOUND = 0x0002
	EROFS_READDIR_DOT_FOUND    = 0x0004

	EROFS_READDIR_ALL_SPECIAL_FOUND = (EROFS_READDIR_DOTDOT_FOUND | EROFS_READDIR_DOT_FOUND)
)

const (
	// fragments.h
	EROFS_PACKED_INODE = "packed_file"
)

const (
	// io.h
	O_BINARY = 0
)

// Error message constants
const (
	// print.h
	EROFS_MSG_MIN = 0
	EROFS_ERR     = 0
	EROFS_WARN    = 2
	EROFS_INFO    = 3
	EROFS_DBG     = 7
	EROFS_MSG_MAX = 9
)

// IOS const.
const (
	// tar.h
	EROFS_IOS_DECODER_NONE    = 0
	EROFS_IOS_DECODER_GZIP    = 1
	EROFS_IOS_DECODER_LIBLZMA = 2
)

// Z-EROFS constants
const (
	EROFS_NAME_LEN = 255

	Z_EROFS_PCLUSTER_MAX_SIZE  uint32 = 1024 * 1024        // maximum supported encoded size of a physical compressed cluster
	Z_EROFS_PCLUSTER_MAX_DSIZE uint32 = (12 * 1024 * 1024) // maximum supported decoded size of a physical compressed cluster

	Z_EROFS_PCLUSTER_MAX_PAGES uint32 = Z_EROFS_PCLUSTER_MAX_SIZE / 4096
	Z_EROFS_NR_INLINE_PCLUSTER uint32 = 1 // # compressed clusters inline in the inode
	Z_EROFS_CLUSTER_MAX_PAGES  uint32 = 4 // Maximum 4 pages in a cluster
)

// EROFS common
const (
	EROFSIVersionMask    = 0x01
	EROFSIDataLayoutMask = 0x07

	EROFSIVersionBit    = 0
	EROFSIDataLayoutBit = 1
	EROFSIAllBit        = 4
	EROFSIAll           = (1 << EROFSIAllBit) - 1

	EROFSChunkFormatBlkBitsMask = 0x001F
	EROFSChunkFormatIndexes     = 0x0020
	EROFSChunkFormatAll         = EROFSChunkFormatBlkBitsMask | EROFSChunkFormatIndexes

	EROFSInodeLayoutCompact  = 0
	EROFSInodeLayoutExtended = 1

	EROFSXattrIndexUser            = 1
	EROFSXattrIndexPosixACLAccess  = 2
	EROFSXattrIndexPosixACLDefault = 3
	EROFSXattrIndexTrusted         = 4
	EROFSXattrIndexLustre          = 5
	EROFSXattrIndexSecurity        = 6

	EROFSXattrLongPrefix     = 0x80
	EROFSXattrLongPrefixMask = 0x7F

	EROFSXattrFilterBits    = 32
	EROFSXattrFilterDefault = math.MaxUint32
	EROFSXattrFilterSeed    = 0x25BBE08F

	EROFSISLOTBITS = 5

	EROFSNullAddr = 0
)

// ZeroFSL
const (
	ZEROFSCompressionLZ4     = 0
	ZEROFSCompressionLZMA    = 1
	ZEROFSCompressionDeflate = 2
	ZEROFSCompressionZSTD    = 3
	ZEROFSCompressionMax     = 4

	ZEROFSAdviseCompacted2B        = 0x0001
	ZEROFSAdviseBigPCluster1       = 0x0002
	ZEROFSAdviseBigPCluster2       = 0x0004
	ZEROFSAdviseInlinePCluster     = 0x0008
	ZEROFSAdviseInterlacedPCluster = 0x0010
	ZEROFSAdviseFragmentPCluster   = 0x0020

	ZEROFSLClusterTypePlain   = 0
	ZEROFSLClusterTypeHead1   = 1
	ZEROFSLClusterTypeNonHead = 2
	ZEROFSLClusterTypeHead2   = 3
	ZEROFSLClusterTypeMax     = 4
	ZEROFSFragmentInodeBit    = 7

	ZEROFSLILClusterTypeMask = ZEROFSLClusterTypeMax - 1
	ZEROFSLIPartialRef       = 1 << 15
	ZEROFSLID0CblkCnt        = 1 << 11
)

// Compression constants
const (
	// Compression algorithm identifiers
	EROFS_COMPRESSION_LZ4     uint8 = 0
	EROFS_COMPRESSION_DEFLATE uint8 = 1
	EROFS_COMPRESSION_LZ4HC   uint8 = 2
	EROFS_COMPRESSION_LZMA    uint8 = 3
)

// Data import mode constants
const (
	EROFS_MKFS_DATA_IMPORT_DEFAULT  = 0 // Default data import mode
	EROFS_MKFS_DATA_IMPORT_FULLDATA = 1 // Full data import mode
	EROFS_MKFS_DATA_IMPORT_RVSP     = 2 // RVSP data import mode
	EROFS_MKFS_DATA_IMPORT_SPARSE   = 3 // Sparse data import mode
)

// cache const.
const (
	// Cache.h
	DATA  = 0
	META  = 1
	INODE = 2
	DIRA  = 3
	XATTR = 4
	DEVT  = 5
)

// EROFS fragment size constants
const (
	EROFS_FRAGMENT_INMEM_SZ_MAX = 256 * 1024
	EROFS_TOF_HASHLEN           = 16

	// Fragment hash constants
	FRAGMENT_HASHSIZE = 65536
)

// Encoding of the file mode.
const (
	S_IFMT = 0170000 /* These bits determine file type.  */

	/* File types.  */
	S_IFDIR  = 0040000 /* Directory.  */
	S_IFCHR  = 0020000 /* Character device.  */
	S_IFBLK  = 0060000 /* Block device.  */
	S_IFREG  = 0100000 /* Regular file.  */
	S_IFIFO  = 0010000 /* FIFO.  */
	S_IFLNK  = 0120000 /* Symbolic link.  */
	S_IFSOCK = 0140000 /* Socket.  */
)

// Flags for getRandom
const (
	GRND_NONBLOCK = 0x01
	GRND_RANDOM   = 0x02
	GRND_INSECURE = 0x04
)

const (
	Z_EROFS_COMPRESSION_LZ4 = iota
	Z_EROFS_COMPRESSION_LZMA
	Z_EROFS_COMPRESSION_DEFLATE
	Z_EROFS_COMPRESSION_ZSTD
	Z_EROFS_COMPRESSION_MAX
)

const (
	LZ4_DISTANCE_MAX = 65535
)

const (
	NR_HARDLINK_HASHTABLE      = 16384
	Z_EROFS_LZMA_MAX_DICT_SIZE = (8 * Z_EROFS_PCLUSTER_MAX_SIZE)
	Z_EROFS_ZSTD_MAX_DICT_SIZE = Z_EROFS_PCLUSTER_MAX_SIZE
)

// FRAGMENT_HASH computes the hash for a fragment
func FRAGMENT_HASH(c uint) uint {
	return c & (FRAGMENT_HASHSIZE - 1)
}
