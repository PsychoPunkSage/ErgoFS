package types

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
}

// CompressionOption represents compression settings
type CompressionOption struct {
	Algorithm string
	Level     int
	DictSize  uint32
}

// DefaultConfig returns a default configuration
func DefaultConfig() *Config {
	return &Config{
		DebugLevel:           5,
		MountPoint:           "/",
		InlineData:           true,
		XattrNameFilter:      true,
		ShowProgress:         true,
		InlineXattrTolerance: 2,
	}
}

// Global configuration instance
var GlobalConfig = DefaultConfig()
