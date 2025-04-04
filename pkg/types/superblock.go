package types

import (
	"bytes"
	"encoding/binary"
	"hash/crc32"
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

// SuperBlockOnDisk is the on-disk representation with explicit little-endian encoding
type SuperBlockOnDisk struct {
	Magic               [4]byte  // __le32
	Checksum            [4]byte  // __le32
	FeatureCompat       [4]byte  // __le32
	BlkSzBits           uint8    // __u8
	SbExtSlots          uint8    // __u8
	RootNid             [2]byte  // __le16
	Inos                [8]byte  // __le64
	BuildTime           [8]byte  // __le64
	BuildTimeNsec       [4]byte  // __le32
	Blocks              [4]byte  // __le32
	MetaBlkAddr         [4]byte  // __le32
	XattrBlkAddr        [4]byte  // __le32
	UUID                [16]byte // __u8[16]
	VolumeName          [16]byte // __u8[16]
	FeatureIncompat     [4]byte  // __le32
	U1                  [2]byte  // Union: __le16 available_compr_algs or __le16 lz4_max_distance
	ExtraDevices        [2]byte  // __le16
	DevtSlotOff         [2]byte  // __le16
	DirBlkBits          uint8    // __u8
	XattrPrefixCount    uint8    // __u8
	XattrPrefixStart    [4]byte  // __le32
	PackedNid           [8]byte  // __le64
	XattrFilterReserved uint8    // __u8
	Reserved2           [23]byte // __u8[23]
}

// FromDisk converts the on-disk representation to an in-memory SuperBlock
func FromDisk(diskSb *SuperBlockOnDisk) *SuperBlock {
	sb := &SuperBlock{}

	// Convert all fields from little-endian
	sb.Magic = binary.LittleEndian.Uint32(diskSb.Magic[:])
	sb.Checksum = binary.LittleEndian.Uint32(diskSb.Checksum[:])
	sb.FeatureCompat = binary.LittleEndian.Uint32(diskSb.FeatureCompat[:])
	sb.BlkSzBits = diskSb.BlkSzBits
	sb.SbExtSlots = diskSb.SbExtSlots
	sb.RootNid = binary.LittleEndian.Uint16(diskSb.RootNid[:])
	sb.Inos = binary.LittleEndian.Uint64(diskSb.Inos[:])
	sb.BuildTime = binary.LittleEndian.Uint64(diskSb.BuildTime[:])
	sb.BuildTimeNsec = binary.LittleEndian.Uint32(diskSb.BuildTimeNsec[:])
	sb.Blocks = binary.LittleEndian.Uint32(diskSb.Blocks[:])
	sb.MetaBlkAddr = binary.LittleEndian.Uint32(diskSb.MetaBlkAddr[:])
	sb.XattrBlkAddr = binary.LittleEndian.Uint32(diskSb.XattrBlkAddr[:])
	copy(sb.UUID[:], diskSb.UUID[:])
	copy(sb.VolumeName[:], diskSb.VolumeName[:])
	sb.FeatureIncompat = binary.LittleEndian.Uint32(diskSb.FeatureIncompat[:])

	// Convert union field - store in both fields
	unionValue := binary.LittleEndian.Uint16(diskSb.U1[:])
	sb.CompressInfo = unionValue

	sb.ExtraDevices = binary.LittleEndian.Uint16(diskSb.ExtraDevices[:])
	sb.DevtSlotOff = binary.LittleEndian.Uint16(diskSb.DevtSlotOff[:])
	sb.DirBlkBits = diskSb.DirBlkBits
	sb.XattrPrefixCount = diskSb.XattrPrefixCount
	sb.XattrPrefixStart = binary.LittleEndian.Uint32(diskSb.XattrPrefixStart[:])
	sb.PackedNid = binary.LittleEndian.Uint64(diskSb.PackedNid[:])
	sb.XattrFilterReserved = diskSb.XattrFilterReserved
	copy(sb.Reserved2[:], diskSb.Reserved2[:])

	return sb
}

// ToDisk converts the in-memory SuperBlock to its on-disk representation
func (sb *SuperBlock) ToDisk() *SuperBlockOnDisk {
	diskSb := &SuperBlockOnDisk{}

	// Convert all fields to little-endian
	binary.LittleEndian.PutUint32(diskSb.Magic[:], sb.Magic)
	binary.LittleEndian.PutUint32(diskSb.Checksum[:], sb.Checksum)
	binary.LittleEndian.PutUint32(diskSb.FeatureCompat[:], sb.FeatureCompat)
	diskSb.BlkSzBits = sb.BlkSzBits
	diskSb.SbExtSlots = sb.SbExtSlots
	binary.LittleEndian.PutUint16(diskSb.RootNid[:], sb.RootNid)
	binary.LittleEndian.PutUint64(diskSb.Inos[:], sb.Inos)
	binary.LittleEndian.PutUint64(diskSb.BuildTime[:], sb.BuildTime)
	binary.LittleEndian.PutUint32(diskSb.BuildTimeNsec[:], sb.BuildTimeNsec)
	binary.LittleEndian.PutUint32(diskSb.Blocks[:], sb.Blocks)
	binary.LittleEndian.PutUint32(diskSb.MetaBlkAddr[:], sb.MetaBlkAddr)
	binary.LittleEndian.PutUint32(diskSb.XattrBlkAddr[:], sb.XattrBlkAddr)
	copy(diskSb.UUID[:], sb.UUID[:])
	copy(diskSb.VolumeName[:], sb.VolumeName[:])
	binary.LittleEndian.PutUint32(diskSb.FeatureIncompat[:], sb.FeatureIncompat)

	// Handle the union field based on compression config
	// if sb.HasCompressionConfig() {
	binary.LittleEndian.PutUint16(diskSb.U1[:], sb.CompressInfo)
	// } else {
	// binary.LittleEndian.PutUint16(diskSb.U1[:], sb.Lz4MaxDistance)
	// }

	binary.LittleEndian.PutUint16(diskSb.ExtraDevices[:], sb.ExtraDevices)
	binary.LittleEndian.PutUint16(diskSb.DevtSlotOff[:], sb.DevtSlotOff)
	diskSb.DirBlkBits = sb.DirBlkBits
	diskSb.XattrPrefixCount = sb.XattrPrefixCount
	binary.LittleEndian.PutUint32(diskSb.XattrPrefixStart[:], sb.XattrPrefixStart)
	binary.LittleEndian.PutUint64(diskSb.PackedNid[:], sb.PackedNid)
	diskSb.XattrFilterReserved = sb.XattrFilterReserved
	copy(diskSb.Reserved2[:], sb.Reserved2[:])

	return diskSb
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
