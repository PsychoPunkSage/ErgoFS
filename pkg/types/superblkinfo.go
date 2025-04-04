package types

import (
	"fmt"
	"math"
	"time"
	"unsafe"

	errs "github.com/PsychoPunkSage/ErgoFS/pkg/errors"
)

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
	ISlotBits uint8 // unsigned char
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

	BDev     *ErofsVFile
	DevBlkSz int
	Devsz    uint64
	DevT     uint64

	// Blob information
	NBlobs uint32
	BlobFd [256]uint32

	// Buffer manager
	Bmgr        *BufferManager
	PackedInode *ErofsPackedInode

	// Deduplication stats
	SavedByDeduplication uint64

	// Useqpl flag
	UseQpl bool
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

// Initialize a new SuperBlockInfo with default values
func NewSuperBlockInfo() *SuperBlkInfo {
	sbi := &SuperBlkInfo{
		BlkSzBits:       uint8(math.Log2(float64(EROFS_MAX_BLOCK_SIZE))),
		FeatureIncompat: EROFS_FEATURE_INCOMPAT_ZERO_PADDING,
		FeatureCompat:   EROFS_FEATURE_COMPAT_SB_CHKSUM | EROFS_FEATURE_COMPAT_MTIME,
		ISlotBits:       5, // EROFS_ISLOTBITS
		TotalBlocks:     1, // Start with 1 for the superblock itself
	}

	// Generate a random UUID
	UUIDGenerate(sbi.UUID[:])

	return sbi
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
func (sbi *SuperBlkInfo) ErofsBlockSize() uint64 {
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
func ErofsEnableSbChksum(sbi *SuperBlkInfo, crc *uint32) int {
	var ret int
	var buf [EROFS_MAX_BLOCK_SIZE]byte
	var length uint32
	var sb *SuperBlock

	ret = ErofsBlkRead(sbi, 0, buf[:], 0, uint32(ErofsBlknr(sbi, uint(EROFS_SUPER_END))+1))
	fmt.Printf("ret: %v\n", ret)
	if ret != 0 {
		// ErofsErr("failed to read superblock to set checksum: %s",
		// ErofsStrerror(ret))
		return ret
	}

	/*
	 * skip the first 1024 bytes, to allow for the installation
	 * of x86 boot sectors and other oddities.
	 */
	sb = (*SuperBlock)(unsafe.Pointer(&buf[EROFS_SUPER_OFFSET]))

	if Le32ToCpu(sb.Magic) != EROFS_SUPER_MAGIC_V1 {
		// ErofsErr("internal error: not an erofs valid image")
		return -errs.EFAULT
	}

	/* turn on checksum feature */
	sb.FeatureCompat = CpuToLe32(Le32ToCpu(sb.FeatureCompat) |
		EROFS_FEATURE_COMPAT_SB_CHKSUM)
	if ErofsBlkSiz(sbi) > EROFS_SUPER_OFFSET {
		length = ErofsBlkSiz(sbi) - EROFS_SUPER_OFFSET
	} else {
		length = ErofsBlkSiz(sbi)
	}
	*crc = Crc32c(^uint32(0), (*[1<<31 - 1]byte)(unsafe.Pointer(sb))[:length])

	/* set up checksum field to erofs_super_block */
	sb.Checksum = CpuToLe32(*crc)

	ret = ErofsBlkWrite(sbi, buf[:], 0, 1)
	if ret != 0 {
		// ErofsErr("failed to write checksummed superblock: %s",
		// ErofsStrerror(ret))
		return ret
	}

	return 0
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

func ErofsBlkSiz(sbi *SuperBlkInfo) uint32 {
	return 1 << sbi.BlkSzBits
}
