package types

import (
	"bytes"
	"crypto/rand"
	"encoding/binary"
	"hash/crc32"
	"math"
	"time"
)

// SuperBlock represents the on-disk EROFS superblock structure
type SuperBlock struct {
	Magic           uint32   // Magic number 0xE0F5E1E2
	Checksum        uint32   // crc32c to avoid unexpected on-disk overlap
	FeatureCompat   uint32   // Compatible features
	BlkSzBits       uint8    // Filesystem block size in bit shift
	SbExtSlots      uint8    // Superblock size = 128 + sb_extslots * 16
	RootNid         uint16   // Nid of root directory
	Inos            uint64   // Total valid inode numbers
	BuildTime       uint64   // Compact inode time derivation
	BuildTimeNsec   uint32   // Compact inode time derivation in ns scale
	Blocks          uint32   // Used for statfs
	MetaBlkAddr     uint32   // Start block address of metadata area
	XattrBlkAddr    uint32   // Start block address of shared xattr area
	UUID            [16]byte // 128-bit UUID for volume
	VolumeName      [16]byte // Volume name
	FeatureIncompat uint32   // Incompatible features
	// Union field for LZ4 max distance or available compression algorithms
	// We'll use CompressInfo for both cases
	CompressInfo        uint16
	ExtraDevices        uint16   // Number of devices besides the primary device
	DevtSlotOff         uint16   // Start offset = devt_slotoff * devt_slotsize
	DirBlkBits          uint8    // Directory block size in bit shift
	XattrPrefixCount    uint8    // Number of long xattr name prefixes
	XattrPrefixStart    uint32   // Start of long xattr prefixes
	PackedNid           uint64   // Nid of the special packed inode
	XattrFilterReserved uint8    // Reserved for xattr name filter
	Reserved2           [23]byte // Reserved space
}

// DeviceSlot represents a slot in the device table
type DeviceSlot struct {
	Mapped   uint32   // Mapped blkaddr of the device
	Blocks   uint32   // Total block count of the device
	Reserved [8]byte  // Reserved for extension
	Tag      [16]byte // Human readable tag (string or UUID)
}

// // BlockSize returns the block size of the filesystem
// func (sb *SuperBlock) BlockSize() uint32 {
// 	return 1 << sb.BlocksizeIlog
// }

// SetFeatureCompat sets a compatible feature flag
func (sb *SuperBlock) SetFeatureCompat(feature uint32) {
	sb.FeatureCompat |= feature
}

// ClearFeatureCompat clears a compatible feature flag
func (sb *SuperBlock) ClearFeatureCompat(feature uint32) {
	sb.FeatureCompat &= ^feature
}

// HasFeatureCompat checks if a compatible feature is enabled
func (sb *SuperBlock) HasFeatureCompat(feature uint32) bool {
	return (sb.FeatureCompat & feature) != 0
}

// SetFeatureIncompat sets an incompatible feature flag
func (sb *SuperBlock) SetFeatureIncompat(feature uint32) {
	sb.FeatureIncompat |= feature
}

// ClearFeatureIncompat clears an incompatible feature flag
func (sb *SuperBlock) ClearFeatureIncompat(feature uint32) {
	sb.FeatureIncompat &= ^feature
}

// HasFeatureIncompat checks if an incompatible feature is enabled
func (sb *SuperBlock) HasFeatureIncompat(feature uint32) bool {
	return (sb.FeatureIncompat & feature) != 0
}

// CalculateChecksum calculates the CRC32C checksum for the superblock
func (sb *SuperBlock) CalculateChecksum() uint32 {
	// Save the current checksum and set it to 0 for calculation
	oldChecksum := sb.Checksum
	sb.Checksum = 0

	// Serialize the superblock
	buf := new(bytes.Buffer)
	binary.Write(buf, binary.LittleEndian, sb)

	// Calculate the checksum
	checksum := crc32.ChecksumIEEE(buf.Bytes())

	// Restore the original checksum
	sb.Checksum = oldChecksum

	return checksum
}

// SetChecksum sets the checksum in the superblock
func (sb *SuperBlock) SetChecksum() {
	sb.Checksum = sb.CalculateChecksum()
}

// ValidateChecksum validates the checksum in the superblock
func (sb *SuperBlock) ValidateChecksum() bool {
	return sb.Checksum == sb.CalculateChecksum()
}

// SuperBlockFromInfo creates a SuperBlock from a SuperBlkInfo
func SuperBlockFromInfo(info *SuperBlkInfo) *SuperBlock {
	sb := &SuperBlock{
		Magic:            EROFS_SUPER_MAGIC_V1,
		FeatureCompat:    info.FeatureCompat,
		FeatureIncompat:  info.FeatureIncompat,
		BlkSzBits:        info.BlkSzBits,
		SbExtSlots:       0, // Default value
		RootNid:          uint16(info.RootNid),
		Inos:             info.Inos,
		BuildTime:        uint64(info.BuildTime),
		BuildTimeNsec:    info.BuildTimeNsec,
		Blocks:           uint32(info.TotalBlocks),
		MetaBlkAddr:      info.MetaBlkAddr,
		XattrBlkAddr:     info.XattrBlkAddr,
		CompressInfo:     info.AvailableComprAlgs, // to remove error
		ExtraDevices:     info.ExtraDevices,
		DirBlkBits:       info.BlkSzBits, // Use same as block size by default
		XattrPrefixCount: info.XattrPrefixCount,
		XattrPrefixStart: info.XattrPrefixStart,
		PackedNid:        info.PackedNid,
	}

	// Set checksum if enabled
	if info.FeatureCompat&EROFS_FEATURE_COMPAT_SB_CHKSUM != 0 {
		sb.SetChecksum()
	}

	return sb
}

// SuperBlkInfo represents the superblock information
// This corresponds to struct erofs_sb_info in the original codebase
type SuperBlkInfo struct {
	// LZ4 compression info
	Lz4 struct {
		MaxDistance     uint16
		MaxPclusterBlks uint16
	}

	// Device information
	Devs    []DeviceInfo
	DevName string

	// Block counts
	TotalBlocks         uint64
	PrimaryDeviceBlocks uint64

	// Block addresses
	MetaBlkAddr  uint32
	XattrBlkAddr uint32

	// Feature flags
	FeatureCompat   uint32
	FeatureIncompat uint32

	// Block size info
	ISlotBits uint8
	BlkSzBits uint8

	// Superblock metadata
	SbSize        uint32
	BuildTimeNsec uint32
	BuildTime     uint64

	// Root information
	RootNid uint32
	Inos    uint64

	// UUID and volume info
	UUID       [16]byte
	VolumeName [16]byte

	// Checksum
	Checksum uint32

	// Compression algorithms
	AvailableComprAlgs uint16

	// Device info
	ExtraDevices uint16
	DevtSlotOff  uint16
	DeviceIdMask uint16

	// Packed inode info
	PackedNid uint64

	// Xattr information
	XattrPrefixStart uint32
	XattrPrefixCount uint8
	XattrPrefixes    []XattrPrefixItem

	// Blob information
	NBlobs uint32
	BlobFd [256]uint32

	// Buffer manager
	Bmgr        *BufferManager
	PackedInode *PackedInode

	// Deduplication stats
	SavedByDeduplication uint64

	// Useqpl flag
	UseQpl bool
}

// DeviceInfo represents information about a device in a multi-device setup
type DeviceInfo struct {
	Tag           [64]byte
	Blocks        uint32
	MappedBlkAddr uint32
}

// XattrPrefixItem holds extended attribute prefix information <PPS:: See in C>
type XattrPrefixItem struct {
	Prefix   *XattrLongPrefix
	InfixLen uint8
}

// XattrLongPrefix represents a long extended attribute prefix <PPS:: See in C>
type XattrLongPrefix struct {
	// Add fields as needed for your implementation
}

// PackedInode represents a packed inode <PPS:: See in C>
type PackedInode struct {
	// Add fields as needed for your implementation
}

// BufferManager manages buffer blocks
type BufferManager struct {
	Sbi             *SuperBlkInfo
	MappedBuckets   [][][]ListHead
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

// ListHead represents a linked list head
type ListHead struct {
	Prev *ListHead
	Next *ListHead
}

// Initialize a new SuperBlockInfo with default values
func NewSuperBlockInfo() *SuperBlkInfo {
	sbi := &SuperBlkInfo{
		BlkSzBits:       uint8(math.Log2(float64(EROFS_MAX_BLOCK_SIZE))),
		FeatureIncompat: EROFS_FEATURE_INCOMPAT_ZERO_PADDING,
		FeatureCompat:   EROFS_FEATURE_COMPAT_SB_CHKSUM | EROFS_FEATURE_COMPAT_MTIME,
		ISlotBits:       5, // EROFS_ISLOTBITS
	}

	// Generate a random UUID
	GenerateUUID(sbi.UUID[:])

	return sbi
}

// GenerateUUID generates a random UUID
func GenerateUUID(out []byte) {
	newUUID := make([]byte, 16)
	_, err := rand.Read(newUUID)
	if err != nil {
		panic("Failed to generate random UUID")
	}

	// Set UUID version (4) and variant (RFC4122)
	newUUID[6] = (newUUID[6] & 0x0F) | 0x40 // Version 4
	newUUID[8] = (newUUID[8] & 0x3F) | 0x80 // Variant RFC4122

	copy(out, newUUID)
}

// SetTimestamp sets the build time to the current time
func (sbi *SuperBlkInfo) SetTimestamp() {
	now := time.Now()
	sbi.BuildTime = uint64(now.Unix())
	sbi.BuildTimeNsec = uint32(now.Nanosecond())
}

// SetCustomTimestamp sets a custom build time
func (sbi *SuperBlkInfo) SetCustomTimestamp(timestamp uint64) {
	sbi.BuildTime = timestamp
	sbi.BuildTimeNsec = 0
}

// ErofsBlockSize returns the block size in bytes
func (sbi *SuperBlkInfo) ErofsBlockSize() uint32 {
	return 1 << sbi.BlkSzBits
}

// WriteSuperblock writes the superblock to the provided buffer
func (sbi *SuperBlkInfo) WriteSuperblock() ([]byte, error) {
	// Calculate the superblock block size (rounded up to block size)
	sbBlkSize := int(sbi.ErofsBlockSize())

	// Create a buffer for the superblock
	buf := make([]byte, sbBlkSize)

	Debug(EROFS_DBG, "Preparing superblock with block size %d", sbBlkSize)

	// Create the superblock structure
	sb := &SuperBlock{
		Magic:            EROFS_SUPER_MAGIC_V1,
		BlkSzBits:        sbi.BlkSzBits,
		RootNid:          uint16(sbi.RootNid),
		Inos:             sbi.Inos,
		BuildTime:        sbi.BuildTime,
		BuildTimeNsec:    sbi.BuildTimeNsec,
		MetaBlkAddr:      sbi.MetaBlkAddr,
		XattrBlkAddr:     sbi.XattrBlkAddr,
		XattrPrefixCount: sbi.XattrPrefixCount,
		XattrPrefixStart: sbi.XattrPrefixStart,
		FeatureIncompat:  sbi.FeatureIncompat,
		FeatureCompat:    sbi.FeatureCompat & ^uint32(EROFS_FEATURE_COMPAT_SB_CHKSUM),
		ExtraDevices:     sbi.ExtraDevices,
		DevtSlotOff:      sbi.DevtSlotOff,
		PackedNid:        sbi.PackedNid,
		// Set directory block bits to same as filesystem block bits for simplicity
		DirBlkBits: sbi.BlkSzBits,
	}

	// Set blocks
	sb.Blocks = uint32(sbi.TotalBlocks)
	if sb.Blocks == 0 {
		// If blocks not set, default to 2 (superblock + metadata)
		sb.Blocks = 2
	}

	Debug(EROFS_DBG, "Setting superblock with %d blocks", sb.Blocks)

	// Copy UUID and volume name
	copy(sb.UUID[:], sbi.UUID[:])
	copy(sb.VolumeName[:], sbi.VolumeName[:])

	// Set compression info based on feature flags
	if HasFeature(sbi, "compr_cfgs") {
		sb.CompressInfo = sbi.AvailableComprAlgs
		Debug(EROFS_DBG, "Using compression algorithms: 0x%04x", sb.CompressInfo)
	} else {
		sb.CompressInfo = sbi.Lz4.MaxDistance
		Debug(EROFS_DBG, "Using LZ4 max distance: %d", sb.CompressInfo)
	}

	// Convert the superblock to binary
	// We'll do this manually to ensure correct byte ordering (little-endian)

	// Copy the magic number
	buf[EROFS_SUPER_OFFSET+0] = byte(sb.Magic)
	buf[EROFS_SUPER_OFFSET+1] = byte(sb.Magic >> 8)
	buf[EROFS_SUPER_OFFSET+2] = byte(sb.Magic >> 16)
	buf[EROFS_SUPER_OFFSET+3] = byte(sb.Magic >> 24)

	// Copy the checksum (initially zero)
	buf[EROFS_SUPER_OFFSET+4] = 0
	buf[EROFS_SUPER_OFFSET+5] = 0
	buf[EROFS_SUPER_OFFSET+6] = 0
	buf[EROFS_SUPER_OFFSET+7] = 0

	// Feature compatibility flags
	buf[EROFS_SUPER_OFFSET+8] = byte(sb.FeatureCompat)
	buf[EROFS_SUPER_OFFSET+9] = byte(sb.FeatureCompat >> 8)
	buf[EROFS_SUPER_OFFSET+10] = byte(sb.FeatureCompat >> 16)
	buf[EROFS_SUPER_OFFSET+11] = byte(sb.FeatureCompat >> 24)

	// Block size bits
	buf[EROFS_SUPER_OFFSET+12] = sb.BlkSzBits

	// Superblock extension slots
	buf[EROFS_SUPER_OFFSET+13] = sb.SbExtSlots

	// Root inode NID
	buf[EROFS_SUPER_OFFSET+14] = byte(sb.RootNid)
	buf[EROFS_SUPER_OFFSET+15] = byte(sb.RootNid >> 8)

	// Inode count
	buf[EROFS_SUPER_OFFSET+16] = byte(sb.Inos)
	buf[EROFS_SUPER_OFFSET+17] = byte(sb.Inos >> 8)
	buf[EROFS_SUPER_OFFSET+18] = byte(sb.Inos >> 16)
	buf[EROFS_SUPER_OFFSET+19] = byte(sb.Inos >> 24)
	buf[EROFS_SUPER_OFFSET+20] = byte(sb.Inos >> 32)
	buf[EROFS_SUPER_OFFSET+21] = byte(sb.Inos >> 40)
	buf[EROFS_SUPER_OFFSET+22] = byte(sb.Inos >> 48)
	buf[EROFS_SUPER_OFFSET+23] = byte(sb.Inos >> 56)

	// Build time
	buf[EROFS_SUPER_OFFSET+24] = byte(sb.BuildTime)
	buf[EROFS_SUPER_OFFSET+25] = byte(sb.BuildTime >> 8)
	buf[EROFS_SUPER_OFFSET+26] = byte(sb.BuildTime >> 16)
	buf[EROFS_SUPER_OFFSET+27] = byte(sb.BuildTime >> 24)
	buf[EROFS_SUPER_OFFSET+28] = byte(sb.BuildTime >> 32)
	buf[EROFS_SUPER_OFFSET+29] = byte(sb.BuildTime >> 40)
	buf[EROFS_SUPER_OFFSET+30] = byte(sb.BuildTime >> 48)
	buf[EROFS_SUPER_OFFSET+31] = byte(sb.BuildTime >> 56)

	// Build time nsec
	buf[EROFS_SUPER_OFFSET+32] = byte(sb.BuildTimeNsec)
	buf[EROFS_SUPER_OFFSET+33] = byte(sb.BuildTimeNsec >> 8)
	buf[EROFS_SUPER_OFFSET+34] = byte(sb.BuildTimeNsec >> 16)
	buf[EROFS_SUPER_OFFSET+35] = byte(sb.BuildTimeNsec >> 24)

	// Blocks
	buf[EROFS_SUPER_OFFSET+36] = byte(sb.Blocks)
	buf[EROFS_SUPER_OFFSET+37] = byte(sb.Blocks >> 8)
	buf[EROFS_SUPER_OFFSET+38] = byte(sb.Blocks >> 16)
	buf[EROFS_SUPER_OFFSET+39] = byte(sb.Blocks >> 24)

	// Meta block address
	buf[EROFS_SUPER_OFFSET+40] = byte(sb.MetaBlkAddr)
	buf[EROFS_SUPER_OFFSET+41] = byte(sb.MetaBlkAddr >> 8)
	buf[EROFS_SUPER_OFFSET+42] = byte(sb.MetaBlkAddr >> 16)
	buf[EROFS_SUPER_OFFSET+43] = byte(sb.MetaBlkAddr >> 24)

	// Xattr block address
	buf[EROFS_SUPER_OFFSET+44] = byte(sb.XattrBlkAddr)
	buf[EROFS_SUPER_OFFSET+45] = byte(sb.XattrBlkAddr >> 8)
	buf[EROFS_SUPER_OFFSET+46] = byte(sb.XattrBlkAddr >> 16)
	buf[EROFS_SUPER_OFFSET+47] = byte(sb.XattrBlkAddr >> 24)

	// UUID
	copy(buf[EROFS_SUPER_OFFSET+48:EROFS_SUPER_OFFSET+64], sb.UUID[:])

	// Volume name
	copy(buf[EROFS_SUPER_OFFSET+64:EROFS_SUPER_OFFSET+80], sb.VolumeName[:])

	// Feature incompatibility flags
	buf[EROFS_SUPER_OFFSET+80] = byte(sb.FeatureIncompat)
	buf[EROFS_SUPER_OFFSET+81] = byte(sb.FeatureIncompat >> 8)
	buf[EROFS_SUPER_OFFSET+82] = byte(sb.FeatureIncompat >> 16)
	buf[EROFS_SUPER_OFFSET+83] = byte(sb.FeatureIncompat >> 24)

	// Compression info
	buf[EROFS_SUPER_OFFSET+84] = byte(sb.CompressInfo)
	buf[EROFS_SUPER_OFFSET+85] = byte(sb.CompressInfo >> 8)

	// Extra devices
	buf[EROFS_SUPER_OFFSET+86] = byte(sb.ExtraDevices)
	buf[EROFS_SUPER_OFFSET+87] = byte(sb.ExtraDevices >> 8)

	// Device table slot offset
	buf[EROFS_SUPER_OFFSET+88] = byte(sb.DevtSlotOff)
	buf[EROFS_SUPER_OFFSET+89] = byte(sb.DevtSlotOff >> 8)

	// Directory block bits
	buf[EROFS_SUPER_OFFSET+90] = sb.DirBlkBits

	// Xattr prefix count
	buf[EROFS_SUPER_OFFSET+91] = sb.XattrPrefixCount

	// Xattr prefix start
	buf[EROFS_SUPER_OFFSET+92] = byte(sb.XattrPrefixStart)
	buf[EROFS_SUPER_OFFSET+93] = byte(sb.XattrPrefixStart >> 8)
	buf[EROFS_SUPER_OFFSET+94] = byte(sb.XattrPrefixStart >> 16)
	buf[EROFS_SUPER_OFFSET+95] = byte(sb.XattrPrefixStart >> 24)

	// Packed NID
	buf[EROFS_SUPER_OFFSET+96] = byte(sb.PackedNid)
	buf[EROFS_SUPER_OFFSET+97] = byte(sb.PackedNid >> 8)
	buf[EROFS_SUPER_OFFSET+98] = byte(sb.PackedNid >> 16)
	buf[EROFS_SUPER_OFFSET+99] = byte(sb.PackedNid >> 24)
	buf[EROFS_SUPER_OFFSET+100] = byte(sb.PackedNid >> 32)
	buf[EROFS_SUPER_OFFSET+101] = byte(sb.PackedNid >> 40)
	buf[EROFS_SUPER_OFFSET+102] = byte(sb.PackedNid >> 48)
	buf[EROFS_SUPER_OFFSET+103] = byte(sb.PackedNid >> 56)

	// Xattr filter reserved
	buf[EROFS_SUPER_OFFSET+104] = sb.XattrFilterReserved

	// Reserved space - zero it out
	for i := 0; i < 23; i++ {
		buf[int(EROFS_SUPER_OFFSET)+105+i] = 0
	}

	Debug(EROFS_DBG, "Superblock prepared successfully")
	return buf, nil
}

// EnableSuperblockChecksum computes and sets the superblock checksum
func (sbi *SuperBlkInfo) EnableSuperblockChecksum(buf []byte) (uint32, error) {
	Debug(EROFS_DBG, "Computing superblock checksum")

	// Enable checksum feature in the buffer
	featureCompat := uint32(buf[EROFS_SUPER_OFFSET+8]) |
		(uint32(buf[EROFS_SUPER_OFFSET+9]) << 8) |
		(uint32(buf[EROFS_SUPER_OFFSET+10]) << 16) |
		(uint32(buf[EROFS_SUPER_OFFSET+11]) << 24)

	featureCompat |= EROFS_FEATURE_COMPAT_SB_CHKSUM

	// Update the feature compatibility flag
	buf[EROFS_SUPER_OFFSET+8] = byte(featureCompat)
	buf[EROFS_SUPER_OFFSET+9] = byte(featureCompat >> 8)
	buf[EROFS_SUPER_OFFSET+10] = byte(featureCompat >> 16)
	buf[EROFS_SUPER_OFFSET+11] = byte(featureCompat >> 24)

	// Clear the current checksum field
	buf[EROFS_SUPER_OFFSET+4] = 0
	buf[EROFS_SUPER_OFFSET+5] = 0
	buf[EROFS_SUPER_OFFSET+6] = 0
	buf[EROFS_SUPER_OFFSET+7] = 0

	// Calculate length for checksum - use one block
	length := int(sbi.ErofsBlockSize())
	if length > int(EROFS_SUPER_OFFSET) {
		length -= int(EROFS_SUPER_OFFSET)
	}

	// Calculate CRC32C checksum
	crc := Crc32c(0xFFFFFFFF, buf[EROFS_SUPER_OFFSET:int(EROFS_SUPER_OFFSET)+length])

	// Update the checksum field
	buf[EROFS_SUPER_OFFSET+4] = byte(crc)
	buf[EROFS_SUPER_OFFSET+5] = byte(crc >> 8)
	buf[EROFS_SUPER_OFFSET+6] = byte(crc >> 16)
	buf[EROFS_SUPER_OFFSET+7] = byte(crc >> 24)

	Debug(EROFS_DBG, "Superblock checksum computed: 0x%08x", crc)
	return crc, nil

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

// HasFeature checks if a specific feature is enabled
func HasFeature(sbi *SuperBlkInfo, feature string) bool {
	switch feature {
	case "lz4_0padding":
		return (sbi.FeatureIncompat & EROFS_FEATURE_INCOMPAT_ZERO_PADDING) != 0
	case "compr_cfgs":
		return (sbi.FeatureIncompat & EROFS_FEATURE_INCOMPAT_COMPR_CFGS) != 0
	case "big_pcluster":
		return (sbi.FeatureIncompat & EROFS_FEATURE_INCOMPAT_BIG_PCLUSTER) != 0
	case "chunked_file":
		return (sbi.FeatureIncompat & EROFS_FEATURE_INCOMPAT_CHUNKED_FILE) != 0
	case "device_table":
		return (sbi.FeatureIncompat & EROFS_FEATURE_INCOMPAT_DEVICE_TABLE) != 0
	case "ztailpacking":
		return (sbi.FeatureIncompat & EROFS_FEATURE_INCOMPAT_ZTAILPACKING) != 0
	case "fragments":
		return (sbi.FeatureIncompat & EROFS_FEATURE_INCOMPAT_FRAGMENTS) != 0
	case "dedupe":
		return (sbi.FeatureIncompat & EROFS_FEATURE_INCOMPAT_DEDUPE) != 0
	case "xattr_prefixes":
		return (sbi.FeatureIncompat & EROFS_FEATURE_INCOMPAT_XATTR_PREFIXES) != 0
	case "sb_chksum":
		return (sbi.FeatureCompat & EROFS_FEATURE_COMPAT_SB_CHKSUM) != 0
	case "xattr_filter":
		return (sbi.FeatureCompat & EROFS_FEATURE_COMPAT_XATTR_FILTER) != 0
	default:
		return false
	}
}

// SetFeature enables a specific feature
func (sbi *SuperBlkInfo) SetFeature(feature string) {
	switch feature {
	case "lz4_0padding":
		sbi.FeatureIncompat |= EROFS_FEATURE_INCOMPAT_ZERO_PADDING
	case "compr_cfgs":
		sbi.FeatureIncompat |= EROFS_FEATURE_INCOMPAT_COMPR_CFGS
	case "big_pcluster":
		sbi.FeatureIncompat |= EROFS_FEATURE_INCOMPAT_BIG_PCLUSTER
	case "chunked_file":
		sbi.FeatureIncompat |= EROFS_FEATURE_INCOMPAT_CHUNKED_FILE
	case "device_table":
		sbi.FeatureIncompat |= EROFS_FEATURE_INCOMPAT_DEVICE_TABLE
	case "ztailpacking":
		sbi.FeatureIncompat |= EROFS_FEATURE_INCOMPAT_ZTAILPACKING
	case "fragments":
		sbi.FeatureIncompat |= EROFS_FEATURE_INCOMPAT_FRAGMENTS
	case "dedupe":
		sbi.FeatureIncompat |= EROFS_FEATURE_INCOMPAT_DEDUPE
	case "xattr_prefixes":
		sbi.FeatureIncompat |= EROFS_FEATURE_INCOMPAT_XATTR_PREFIXES
	case "sb_chksum":
		sbi.FeatureCompat |= EROFS_FEATURE_COMPAT_SB_CHKSUM
	case "xattr_filter":
		sbi.FeatureCompat |= EROFS_FEATURE_COMPAT_XATTR_FILTER
	}
}

// ClearFeature disables a specific feature
func (sbi *SuperBlkInfo) ClearFeature(feature string) {
	switch feature {
	case "lz4_0padding":
		sbi.FeatureIncompat &= ^uint32(EROFS_FEATURE_INCOMPAT_ZERO_PADDING)
	case "compr_cfgs":
		sbi.FeatureIncompat &= ^uint32(EROFS_FEATURE_INCOMPAT_COMPR_CFGS)
	case "big_pcluster":
		sbi.FeatureIncompat &= ^uint32(EROFS_FEATURE_INCOMPAT_BIG_PCLUSTER)
	case "chunked_file":
		sbi.FeatureIncompat &= ^uint32(EROFS_FEATURE_INCOMPAT_CHUNKED_FILE)
	case "device_table":
		sbi.FeatureIncompat &= ^uint32(EROFS_FEATURE_INCOMPAT_DEVICE_TABLE)
	case "ztailpacking":
		sbi.FeatureIncompat &= ^uint32(EROFS_FEATURE_INCOMPAT_ZTAILPACKING)
	case "fragments":
		sbi.FeatureIncompat &= ^uint32(EROFS_FEATURE_INCOMPAT_FRAGMENTS)
	case "dedupe":
		sbi.FeatureIncompat &= ^uint32(EROFS_FEATURE_INCOMPAT_DEDUPE)
	case "xattr_prefixes":
		sbi.FeatureIncompat &= ^uint32(EROFS_FEATURE_INCOMPAT_XATTR_PREFIXES)
	case "sb_chksum":
		sbi.FeatureCompat &= ^uint32(EROFS_FEATURE_COMPAT_SB_CHKSUM)
	case "xattr_filter":
		sbi.FeatureCompat &= ^uint32(EROFS_FEATURE_COMPAT_XATTR_FILTER)
	}
}

// Utility function to round up to the next multiple
func roundUp(value, multiple int) int {
	return ((value + multiple - 1) / multiple) * multiple
}
