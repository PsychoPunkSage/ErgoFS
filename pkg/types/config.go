package types

import "time"

// This file implements configuration structures and functionality similar to
// include/erofs/config.h and lib/config.c in the original codebase

// CompressionOption represents options for a compression algorithm
type CompressionOption struct {
	Algorithm string // Compression algorithm name
	Level     int    // Compression level
	DictSize  uint32 // Dictionary size
}

// Config represents the configuration options for EROFS filesystem creation
// This corresponds to struct erofs_configure in config.h
type Config struct {
	// Basic configuration
	Version           string
	DebugLevel        uint8
	DryRun            bool
	LegacyCompress    bool
	TimeInherit       TimestampType
	ChunkBits         uint8
	InlineData        bool
	ZtailPacking      bool
	Fragments         bool
	AllFragments      bool
	Dedupe            bool
	FragDedupe        FragDedupeMode
	IgnoreMtime       bool
	ShowProgress      bool
	ExtraEANamePrefix bool
	XattrNameFilter   bool
	OvlfsStrip        bool
	HardDereference   bool

	// File paths
	ImagePath                  string
	SourcePath                 string
	BlobDevPath                string
	CompressHintsFile          string
	CompressionOptions         [EROFS_MAX_COMPR_CFGS]CompressionOption
	ForceInodeVersion          ForceInodeVersion
	ForceChunkFormat           ForceChunkFormat
	InlineXattrTolerance       int
	MkfsSegmentSize            uint64
	MtWorkers                  uint32
	MkfsPclusterSizeMax        uint32
	MkfsPclusterSizeDef        uint32
	MkfsPclusterSizePacked     uint32
	MaxDecompressedExtentBytes uint32
	UnixTimestamp              int64
	UserID                     uint32
	GroupID                    uint32
	MountPoint                 string
	UserIDOffset               int64
	GroupIDOffset              int64
	RootXattrIsize             uint32

	// Debug options
	RandomPclusterBlks bool
	RandomAlgorithms   bool
}

// SuperBlkInfo represents the superblock information
// This corresponds to struct erofs_sb_info in the original codebase
type SuperBlkInfo struct {
	// Basic identifiers
	UUID       [16]byte
	VolumeName [16]byte

	// Block and size information
	BlksizeBits      uint8  // Corresponds to blkszbits
	IslotBits        uint8  // Corresponds to islotbits
	TotalBlocks      uint64 // Corresponds to total_blocks
	PrimaryDevBlocks uint64 // Corresponds to primarydevice_blocks
	MetaBlkAddr      uint32 // Corresponds to meta_blkaddr
	XattrBlkAddr     uint32 // Corresponds to xattr_blkaddr

	// Feature flags
	FeatureCompat   uint32 // Feature compatibility flags
	FeatureIncompat uint32 // Feature incompatibility flags

	// Root and inode information
	RootNid uint64 // Corresponds to root_nid
	Inos    uint64 // Corresponds to inos

	// Build info
	BuildTime     int64  // Corresponds to build_time
	BuildTimeNsec uint32 // Corresponds to build_time_nsec
	SbSize        uint32 // Corresponds to sb_size

	// Device and extended information
	ExtraDevices uint16       // Corresponds to extra_devices
	DeviceIdMask uint16       // Corresponds to device_id_mask
	Devices      []DeviceInfo // Corresponds to devs array

	// Xattr information
	XattrPrefixStart uint32 // Corresponds to xattr_prefix_start
	XattrPrefixCount uint8  // Corresponds to xattr_prefix_count

	// Compression information
	AvailableComprAlgs uint16 // Corresponds to available_compr_algs
	PackedNid          uint64 // Corresponds to packed_nid

	// Checksum
	Checksum uint32 // Corresponds to checksum

	// Statistics
	SavedByDeduplication uint64 // Corresponds to saved_by_deduplication
}

// DeviceInfo represents information about a device in a multi-device setup
type DeviceInfo struct {
	Blocks uint32
	Tag    [16]byte
}

// DefaultConfig returns a new Config with default settings
func DefaultConfig() *Config {
	cfg := &Config{
		Version:                    "1.0.0",
		DebugLevel:                 EROFS_WARN,
		DryRun:                     false,
		LegacyCompress:             false,
		InlineData:                 true,
		XattrNameFilter:            true,
		ShowProgress:               true,
		TimeInherit:                TIMESTAMP_UNSPECIFIED,
		UnixTimestamp:              -1,
		UserID:                     ^uint32(0), // -1 in C
		GroupID:                    ^uint32(0), // -1 in C
		InlineXattrTolerance:       2,
		MaxDecompressedExtentBytes: ^uint32(0), // -1 in C
	}

	return cfg
}

// DefaultSuperBlkInfo returns a new SuperBlkInfo with default settings
func DefaultSuperBlkInfo() *SuperBlkInfo {
	now := time.Now()

	sbi := &SuperBlkInfo{
		BlksizeBits:     12, // 4KB default
		FeatureCompat:   EROFS_FEATURE_COMPAT_SB_CHKSUM,
		FeatureIncompat: EROFS_FEATURE_INCOMPAT_ZERO_PADDING,
		BuildTime:       now.Unix(),
		BuildTimeNsec:   uint32(now.Nanosecond()),
	}

	// Generate a random UUID
	sbi.SetRandomUUID()

	return sbi
}

// SetRandomUUID sets a random UUID in the superblock info
func (sbi *SuperBlkInfo) SetRandomUUID() {
	// This would implement UUID generation
	// For now, just set a placeholder
	for i := range sbi.UUID {
		sbi.UUID[i] = byte(i + 1)
	}
}

// EraseConfig clears all configuration variables
func EraseConfig(cfg *Config) {
	*cfg = Config{}
}

// SetFsRoot sets the filesystem root directory
func SetFsRoot(rootdir string) {
	// This would implement the fs root setting
	// For now, it's a placeholder
}

// GetFsPath returns the normalized filesystem path
func GetFsPath(fullpath string) string {
	// This would implement path normalization
	// For now, return the path as-is
	return fullpath
}
