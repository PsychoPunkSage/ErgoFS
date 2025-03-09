package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"os"
)

// SuperBlock represents the on-disk EROFS superblock structure
// Matches the current C struct erofs_super_block
type SuperBlock struct {
	Magic               uint32   // file system magic number
	Checksum            uint32   // crc32c to avoid unexpected on-disk overlap
	FeatureCompat       uint32   // feature compatibility flags
	BlkSzBits           uint8    // filesystem block size in bit shift
	SbExtSlots          uint8    // superblock size = 128 + sb_extslots * 16
	Root_nid            uint16   // nid of root directory
	Inos                uint64   // total valid ino # (== f_files - f_favail)
	BuildTime           uint64   // compact inode time derivation
	BuildTimeNsec       uint32   // compact inode time derivation in ns scale
	Blocks              uint32   // used for statfs
	MetaBlkAddr         uint32   // start block address of metadata area
	XattrBlkAddr        uint32   // start block address of shared xattr area
	UUID                [16]byte // 128-bit uuid for volume
	VolumeName          [16]byte // volume name
	FeatureIncompat     uint32   // feature incompatibility flags
	U1                  uint16   // Union: available_compr_algs or lz4_max_distance
	ExtraDevices        uint16   // # of devices besides the primary device
	DevtSlotoff         uint16   // startoff = devt_slotoff * devt_slotsize
	DirBlkBits          uint8    // directory block size in bit shift
	XattrPrefixCount    uint8    // # of long xattr name prefixes
	XattrPrefixStart    uint32   // start of long xattr prefixes
	PackedNid           uint64   // nid of the special packed inode
	XattrFilterReserved uint8    // reserved for xattr name filter
	Reserved2           [23]byte // reserved for extension
}

const EROFS_SUPER_MAGIC_V1 uint32 = 0xE0F5E1E0
const EROFS_SUPER_OFFSET uint32 = 1024

// Feature flag constants
const (
	EROFS_FEATURE_COMPAT_SB_CHKSUM      uint32 = 0x00000001
	EROFS_FEATURE_INCOMPAT_ZERO_PADDING uint32 = 0x00000001
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: erofs-verify <erofs-image>")
		os.Exit(1)
	}

	imagePath := os.Args[1]

	file, err := os.Open(imagePath)
	if err != nil {
		fmt.Printf("Error opening file: %v\n", err)
		os.Exit(1)
	}
	defer file.Close()

	// Get file size
	fileInfo, err := file.Stat()
	if err != nil {
		fmt.Printf("Error getting file info: %v\n", err)
		os.Exit(1)
	}
	fileSize := fileInfo.Size()
	fmt.Printf("File size: %d bytes\n", fileSize)

	// Read first 4K
	header := make([]byte, 4096)
	n, err := file.Read(header)
	if err != nil {
		fmt.Printf("Error reading file: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Read %d bytes from image file\n", n)

	// Print bytes at superblock offset (1024)
	fmt.Println("\nSuperblock header (first 32 bytes at offset 1024):")
	for i := 1024; i < 1024+32 && i < len(header); i++ {
		if i%16 == 0 {
			fmt.Printf("\n%04x: ", i)
		}
		fmt.Printf("%02x ", header[i])
	}

	// Check magic number at offset 1024
	if len(header) >= int(EROFS_SUPER_OFFSET)+4 {
		magic := binary.LittleEndian.Uint32(header[EROFS_SUPER_OFFSET : EROFS_SUPER_OFFSET+4])
		fmt.Printf("Magic at offset 1024: 0x%08x (expected: 0x%08x)\n", magic, EROFS_SUPER_MAGIC_V1)
		if magic != EROFS_SUPER_MAGIC_V1 {
			fmt.Println("❌ ERROR: Magic number mismatch!")
		} else {
			fmt.Println("✅ Magic number matches!")
		}
	} else {
		fmt.Println("❌ ERROR: File too small to contain a superblock")
		os.Exit(1)
	}

	// Read and parse the superblock
	var sb SuperBlock
	sbSize := binary.Size(sb)
	fmt.Printf("SuperBlock structure size: %d bytes\n", sbSize)

	if len(header) >= int(EROFS_SUPER_OFFSET)+sbSize {
		reader := bytes.NewReader(header[EROFS_SUPER_OFFSET:])
		err = binary.Read(reader, binary.LittleEndian, &sb)
		if err != nil {
			fmt.Printf("❌ ERROR reading superblock: %v\n", err)
			os.Exit(1)
		}

		// Display superblock information
		fmt.Println("\n=== Superblock Information ===")
		fmt.Printf("Magic:               0x%08x\n", sb.Magic)
		fmt.Printf("Checksum:            0x%08x\n", sb.Checksum)
		fmt.Printf("Feature Compat:      0x%08x\n", sb.FeatureCompat)
		fmt.Printf("Feature Incompat:    0x%08x\n", sb.FeatureIncompat)
		fmt.Printf("Block Size:          %d bytes (ilog2: %d)\n", 1<<sb.BlkSzBits, sb.BlkSzBits)
		fmt.Printf("Dir Block Size:      %d bytes (ilog2: %d)\n", 1<<sb.DirBlkBits, sb.DirBlkBits)
		fmt.Printf("Superblock Extslots: %d\n", sb.SbExtSlots)
		fmt.Printf("Root NID:            %d\n", sb.Root_nid)
		fmt.Printf("Inode Count:         %d\n", sb.Inos)
		fmt.Printf("Total Blocks:        %d\n", sb.Blocks)
		fmt.Printf("Meta Block Addr:     %d\n", sb.MetaBlkAddr)
		fmt.Printf("Xattr Block Addr:    %d\n", sb.XattrBlkAddr)
		fmt.Printf("Build Time:          %d.%09d\n", sb.BuildTime, sb.BuildTimeNsec)
		fmt.Printf("UUID:                ")
		for i, b := range sb.UUID {
			fmt.Printf("%02x", b)
			if i == 3 || i == 5 || i == 7 || i == 9 {
				fmt.Printf("-")
			}
		}
		fmt.Println()

		fmt.Printf("Volume Name:         ")
		for _, b := range sb.VolumeName {
			if b == 0 {
				break
			}
			fmt.Printf("%c", b)
		}
		fmt.Println()

		// Check for common issues
		fmt.Println("\n=== EROFS Validation Checks ===")

		errors := 0
		warnings := 0

		// 1. Check root nid
		if sb.Root_nid != 1 {
			fmt.Println("❌ ERROR: Root inode number is not 1. Linux kernel expects root inode to be 1.")
			errors++
		} else {
			fmt.Println("✅ Root inode number is correct (1)")
		}

		// 2. Check meta block address
		if sb.MetaBlkAddr == 0 {
			fmt.Println("❌ ERROR: Meta block address is 0. Must point to valid metadata.")
			errors++
		} else {
			metaOffset := uint64(sb.MetaBlkAddr) * uint64(1<<sb.BlkSzBits)
			fmt.Printf("✅ Meta block address is %d (file offset: 0x%x)\n", sb.MetaBlkAddr, metaOffset)

			// Check if meta block address is reasonable
			if metaOffset >= uint64(fileSize) {
				fmt.Println("❌ ERROR: Meta block address points beyond the end of the file!")
				errors++
			}
		}

		// 3. Check block size
		if sb.BlkSzBits < 9 || sb.BlkSzBits > 12 {
			fmt.Printf("⚠️ WARNING: Block size bits (%d) outside normal range (9-12)\n", sb.BlkSzBits)
			warnings++
		} else {
			fmt.Println("✅ Block size bits within normal range")
		}

		// 4. Check superblock checksum if enabled
		if sb.FeatureCompat&EROFS_FEATURE_COMPAT_SB_CHKSUM != 0 {
			// To properly verify, we'd need to implement the same checksum method as the kernel
			fmt.Println("ℹ️ Superblock checksum is enabled")
		}

		// 5. Check if volume size makes sense
		expectedSize := uint64(sb.Blocks) * uint64(1<<sb.BlkSzBits)
		if expectedSize > uint64(fileSize) {
			fmt.Printf("⚠️ WARNING: Expected volume size (%d bytes) exceeds actual file size (%d bytes)\n",
				expectedSize, fileSize)
			warnings++
		} else {
			fmt.Printf("✅ Volume size consistent with file size (%d blocks, %d bytes)\n",
				sb.Blocks, expectedSize)
		}

		// Final verdict
		fmt.Println("\n=== Summary ===")
		if errors > 0 {
			fmt.Printf("❌ Found %d errors that will prevent mounting\n", errors)
		} else if warnings > 0 {
			fmt.Printf("⚠️ Found %d warnings, but filesystem should be mountable\n", warnings)
		} else {
			fmt.Println("✅ No issues detected! Filesystem should be mountable")
		}

	} else {
		fmt.Println("❌ ERROR: File too small to read complete superblock")
	}
}
