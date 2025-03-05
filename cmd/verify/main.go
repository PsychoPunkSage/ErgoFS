package main

import (
	"encoding/binary"
	"fmt"
	"os"
)

// Constants for EROFS
const (
	EROFS_SUPER_MAGIC_V1 uint32 = 0xE0F5E1E0
)

// SuperBlock structure - must match exactly with the EROFS format
type SuperBlock struct {
	Magic           uint32   // EROFS filesystem magic
	ChecksumAlg     uint8    // Checksum algorithm for metadata
	Reserved        uint8    // Reserved for extension
	FeatureCompat   uint16   // Compatible feature flags
	BlocksizeIlog   uint8    // Block size ilog2
	SbExtVer        uint8    // Superblock extension header version
	FeatureIncompat uint16   // Incompatible feature flags
	UUIDBytes       [16]byte // 128-bit UUID
	VolumeName      [16]byte // Volume label
	InodeCount      uint32   // Total valid inode count
	Blocks          uint32   // Total valid block count
	MetaBlkAddr     uint32   // Meta block start address
	XattrBlkAddr    uint32   // Xattr block start address
	ExtraDevices    uint16   // # of extra devices
	Padding1        uint16   // Padding
	InodeExtraSize  uint8    // Size of inode extra metadata
	XattrExtraSize  uint8    // Size of xattr extra metadata
	Padding2        uint16   // Padding
	BuildTime       uint64   // Build time
	BuildTimeNsec   uint32   // Build time nanosecond part
	Reserved2       uint32   // Reserved for extension
	Checksum        uint32   // CRC32C checksum
}

func main() {
	if len(os.Args) != 2 {
		fmt.Println("Usage: erofs-verify <image-file>")
		os.Exit(1)
	}

	imageFile := os.Args[1]

	// Open the image file
	file, err := os.Open(imageFile)
	if err != nil {
		fmt.Printf("Error opening file: %v\n", err)
		os.Exit(1)
	}
	defer file.Close()

	// Read raw bytes first for debugging
	fmt.Println("Raw Superblock Data (first 32 bytes):")
	rawData := make([]byte, 128)
	_, err = file.ReadAt(rawData, 0)
	if err != nil {
		fmt.Printf("Error reading raw superblock data: %v\n", err)
		os.Exit(1)
	}

	// Print raw bytes for debugging
	for i := 0; i < 32; i++ {
		fmt.Printf("%02x ", rawData[i])
		if (i+1)%16 == 0 {
			fmt.Println()
		}
	}
	fmt.Println()

	// Reset file position
	_, err = file.Seek(0, 0)
	if err != nil {
		fmt.Printf("Error seeking file: %v\n", err)
		os.Exit(1)
	}

	// Read the superblock
	sb := SuperBlock{}
	err = binary.Read(file, binary.LittleEndian, &sb)
	if err != nil {
		fmt.Printf("Error reading superblock: %v\n", err)
		os.Exit(1)
	}

	// Display magic number for debugging
	fmt.Printf("Magic number in hex: 0x%08x\n", sb.Magic)
	fmt.Printf("Expected magic:      0x%08x\n", EROFS_SUPER_MAGIC_V1)

	// Verify the magic number
	if sb.Magic != EROFS_SUPER_MAGIC_V1 {
		fmt.Printf("ERROR: Invalid magic number!\n")

		// Check if bytes are swapped (endianness issue)
		swappedMagic := (sb.Magic&0xFF)<<24 | (sb.Magic&0xFF00)<<8 | (sb.Magic&0xFF0000)>>8 | (sb.Magic&0xFF000000)>>24
		if swappedMagic == EROFS_SUPER_MAGIC_V1 {
			fmt.Printf("NOTE: Magic number matches when byte-swapped. Possible endianness issue.\n")
		}

		fmt.Printf("First 4 bytes as uint32 (LE): 0x%08x\n", binary.LittleEndian.Uint32(rawData[:4]))
		fmt.Printf("First 4 bytes as uint32 (BE): 0x%08x\n", binary.BigEndian.Uint32(rawData[:4]))
	}

	// Display superblock information
	fmt.Println("\nEROFS Image Information:")
	fmt.Printf("Magic:            0x%08x\n", sb.Magic)
	fmt.Printf("Checksum Alg:     %d\n", sb.ChecksumAlg)
	fmt.Printf("Block Size:       %d bytes (ilog2: %d)\n", 1<<sb.BlocksizeIlog, sb.BlocksizeIlog)
	fmt.Printf("SB Extension Ver: %d\n", sb.SbExtVer)
	fmt.Printf("Feature Compat:   0x%04x\n", sb.FeatureCompat)
	fmt.Printf("Feature Incompat: 0x%04x\n", sb.FeatureIncompat)

	fmt.Printf("UUID:             ")
	for _, b := range sb.UUIDBytes {
		fmt.Printf("%02x", b)
	}
	fmt.Println()

	// Extract volume name (as a string)
	volName := string(sb.VolumeName[:])
	for i, b := range sb.VolumeName {
		if b == 0 {
			volName = string(sb.VolumeName[:i])
			break
		}
	}
	fmt.Printf("Volume Name:      %s\n", volName)

	fmt.Printf("Inode Count:      %d\n", sb.InodeCount)
	fmt.Printf("Blocks:           %d\n", sb.Blocks)
	fmt.Printf("Meta Block Addr:  0x%08x\n", sb.MetaBlkAddr)
	fmt.Printf("Xattr Block Addr: 0x%08x\n", sb.XattrBlkAddr)
	fmt.Printf("Extra Devices:    %d\n", sb.ExtraDevices)
	fmt.Printf("Build Time:       %d.%09d\n", sb.BuildTime, sb.BuildTimeNsec)
	fmt.Printf("Checksum:         0x%08x\n", sb.Checksum)

	// Verify image file size is consistent with block count
	fileInfo, err := file.Stat()
	if err != nil {
		fmt.Printf("Error getting file info: %v\n", err)
	} else {
		fileSize := fileInfo.Size()
		blockSize := int64(1) << sb.BlocksizeIlog
		expectedBlocks := uint32((fileSize + blockSize - 1) / blockSize)

		fmt.Printf("\nFile Size:        %d bytes\n", fileSize)
		fmt.Printf("Expected Blocks:  %d\n", expectedBlocks)

		if expectedBlocks != sb.Blocks {
			fmt.Printf("WARNING: Block count mismatch. Superblock says %d blocks, file size suggests %d blocks.\n",
				sb.Blocks, expectedBlocks)
		}
	}

	// Read the first inode location
	if sb.InodeCount > 0 {
		fmt.Println("\nAttempting to read root inode metadata...")
		// Assuming root inode is at block 1
		inodeOffset := int64(1) << sb.BlocksizeIlog
		inodeData := make([]byte, 64) // Read a few bytes of the inode
		_, err = file.ReadAt(inodeData, inodeOffset)
		if err != nil {
			fmt.Printf("Error reading inode data: %v\n", err)
		} else {
			fmt.Println("First 32 bytes of root inode area:")
			for i := 0; i < 32; i++ {
				fmt.Printf("%02x ", inodeData[i])
				if (i+1)%16 == 0 {
					fmt.Println()
				}
			}
		}
	}

	fmt.Println("\nVerification complete!")
}
