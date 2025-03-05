package types

import (
	"bytes"
	"encoding/binary"
	"hash/crc32"
)

// SuperBlock represents the on-disk EROFS superblock structure
// This matches the C structure defined in erofs_fs.h
type SuperBlock struct {
	// The order and size of each field must match exactly with the C struct
	Magic            uint32   // EROFS_SUPER_MAGIC_V1
	ChecksumAlg      uint8    // Checksum algorithm for metadata
	BlocksizeIlog    uint8    // Block size ilog2 (matches blkszbits)
	FeatureCompat    uint32   // Compatible feature flags
	FeatureIncompat  uint32   // Incompatible feature flags
	UUIDBytes        [16]byte // 128-bit UUID
	VolumeName       [16]byte // Volume label
	TotalBlocks      uint64   // Total blocks (matches total_blocks)
	PrimarydevBlocks uint64   // Primary device blocks
	MetaBlkAddr      uint32   // Meta block start address
	XattrBlkAddr     uint32   // Xattr block start address
	IslotBits        uint8    // Island slots bits
	_                [3]byte  // Padding to align
	RootNid          uint64   // Root node ID
	InodeCount       uint64   // Total valid inode count (matches inos)
	BuildTime        int64    // Build time
	BuildTimeNsec    uint32   // Build time nanosecond part
	SbSize           uint32   // Total superblock size
	Checksum         uint32   // CRC32C checksum
	ExtraDevices     uint16   // # of extra devices
	DeviceIdMask     uint16   // Device ID mask
	PackedNid        uint64   // Packed node ID
	XattrPrefixStart uint32   // Xattr prefix start
	XattrPrefixCount uint8    // Xattr prefix count
	_                [3]byte  // Padding to align
	AvailComprAlgs   uint16   // Available compression algorithms
	_                [6]byte  // Padding to align
	SavedByDedup     uint64   // Saved by deduplication
}

// DeviceSlot represents a slot in the device table
type DeviceSlot struct {
	Mapped   uint32   // Mapped blkaddr of the device
	Blocks   uint32   // Total block count of the device
	Reserved [8]byte  // Reserved for extension
	Tag      [16]byte // Human readable tag (string or UUID)
}

// BlockSize returns the block size of the filesystem
func (sb *SuperBlock) BlockSize() uint32 {
	return 1 << sb.BlocksizeIlog
}

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
		ChecksumAlg:      0, // Default to CRC32C
		BlocksizeIlog:    info.BlksizeBits,
		FeatureCompat:    info.FeatureCompat,
		FeatureIncompat:  info.FeatureIncompat,
		UUIDBytes:        info.UUID,
		VolumeName:       info.VolumeName,
		InodeCount:       info.Inos,
		TotalBlocks:      info.TotalBlocks,
		PrimarydevBlocks: info.PrimaryDevBlocks,
		ExtraDevices:     info.ExtraDevices,
		BuildTime:        info.BuildTime,
		BuildTimeNsec:    info.BuildTimeNsec,
		MetaBlkAddr:      0, // Will be set later
		XattrBlkAddr:     0, // Will be set later
		IslotBits:        info.IslotBits,
		RootNid:          info.RootNid,
		DeviceIdMask:     info.DeviceIdMask,
		PackedNid:        info.PackedNid,
		XattrPrefixStart: info.XattrPrefixStart,
		XattrPrefixCount: info.XattrPrefixCount,
		AvailComprAlgs:   info.AvailableComprAlgs,
		SbSize:           info.SbSize,
		SavedByDedup:     info.SavedByDeduplication,
	}

	// Set checksum if enabled
	if info.FeatureCompat&EROFS_FEATURE_COMPAT_SB_CHKSUM != 0 {
		sb.SetChecksum()
	}

	return sb
}

/*
I am still gettin gthis issue
./script.sh

Atatched output

I guess I have to make the struct very precisely .. i.e. exactly as C lang counter part.....

I would suggest you to do that...

My project str
*/
