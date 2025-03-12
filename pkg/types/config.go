package types

import (
	"fmt"
	"math"
	"os"
	"path/filepath"
	"runtime"
)

// Config represents the mkfs build configuration
type Config struct {
	// Debug level
	DebugLevel int

	// Output image path
	ImagePath string

	// Source path
	SourcePath string

	// Mount point
	MountPoint string

	// User/Group IDs
	Uid       int64
	Gid       int64
	UidOffset int64
	GidOffset int64

	DryRun  bool
	Version string // For c_version = PACKAGE_VERSIO

	// Block size and compression
	ForceInodeVersion int
	ForceChunkFormat  int
	InlineData        bool
	LegacyCompress    bool
	ZtailPacking      bool
	Fragments         bool
	AllFragments      bool
	Dedupe            bool
	XattrNameFilter   bool

	// Compression settings
	CompressionOptions    []CompressionOption
	MaxDecompressedExtent uint64

	// Time handling
	TimeInherit   uint8
	UnixTimestamp int64
	IgnoreMtime   bool

	// XAttr settings
	InlineXattrTolerance int
	ExtraEANamePrefixes  bool
	RootXattrIsize       uint64

	// Blob and chunks
	BlobDevPath string
	ChunkBits   uint8

	// Visualization
	ShowProgress bool

	// Additional fields from C struct
	CompressHintsFile string // c_compress_hints_file
	FragmentDedupe    uint8  // c_fragdedupe
	OvlfsStrip        bool   // c_ovlfs_strip
	HardDereference   bool   // c_hard_dereference

	// MT support
	MkfsSegmentSize uint64 // c_mkfs_segment_size
	MtWorkers       uint32 // c_mt_workers

	// Cluster sizes
	MkfsPclusterSizeMax    uint32 // c_mkfs_pclustersize_max
	MkfsPclusterSizeDef    uint32 // c_mkfs_pclustersize_def
	MkfsPclusterSizePacked uint32 // c_mkfs_pclustersize_packed

	// Android specific
	TargetOutPath string // target_out_path
	FsConfigFile  string // fs_config_file
	BlockListFile string // block_list_file

	// Debug options
	RandomPclusterBlks bool // c_random_pclusterblks
	RandomAlgorithms   bool // c_random_algorithms
}

// CompressionOption represents compression settings
type CompressionOption struct {
	Algorithm string
	Level     int
	DictSize  uint32
}

// DefaultConfig returns a default configuration
func InitConfigure() *Config {
	return &Config{
		DebugLevel:            EROFS_DBG,
		DryRun:                false,
		IgnoreMtime:           false,
		ForceInodeVersion:     0,
		InlineXattrTolerance:  2,
		UnixTimestamp:         -1,
		Uid:                   -1,
		Gid:                   -1,
		MaxDecompressedExtent: ^uint64(0),
		ShowProgress:          isatty(),
	}
}

func DefaultOptions(sbi *SuperBlkInfo) {
	GlobalConfig.ShowProgress = true
	GlobalConfig.LegacyCompress = false
	GlobalConfig.InlineData = true
	GlobalConfig.XattrNameFilter = true

	// For MT_ENABLED equivalent in Go, we'd need to use runtime.NumCPU()
	// Assuming you want to add this functionality:
	GlobalConfig.MtWorkers = uint32(runtime.NumCPU())
	GlobalConfig.MkfsSegmentSize = 16 * 1024 * 1024 // 16 MB

	// Set blocksize bits based on page size or max block size
	pageSize := os.Getpagesize()
	maxBlockSize := int(EROFS_MAX_BLOCK_SIZE)
	if pageSize > maxBlockSize {
		pageSize = maxBlockSize
	}
	sbi.BlkSzBits = uint8(math.Log2(float64(pageSize)))

	// Set cluster sizes
	GlobalConfig.MkfsPclusterSizeMax = 1 << sbi.BlkSzBits
	GlobalConfig.MkfsPclusterSizeDef = GlobalConfig.MkfsPclusterSizeMax

	// Set features
	sbi.FeatureIncompat = EROFS_FEATURE_INCOMPAT_ZERO_PADDING
	sbi.FeatureCompat = EROFS_FEATURE_COMPAT_SB_CHKSUM |
		EROFS_FEATURE_COMPAT_MTIME
}

func ShowProgs(args []string) {
	if GlobalConfig.DebugLevel >= EROFS_WARN {
		programName := filepath.Base(args[0])
		fmt.Printf("%s %s\n", programName, GlobalConfig.Version)
	}
}

// isatty determines if stdout is a TTY (similar to C's isatty function)
func isatty() bool {
	fileInfo, err := os.Stdout.Stat()
	if err != nil {
		return false
	}

	// Check if it's a character device (TTY)
	// This is OS-specific but works on most Unix-like systems
	return (fileInfo.Mode() & os.ModeCharDevice) != 0
}

// Global configuration instance
var GlobalConfig = InitConfigure()
