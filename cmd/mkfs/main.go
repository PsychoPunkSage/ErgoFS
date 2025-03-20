// package main

// import (
// 	"flag"
// 	"fmt"
// 	"os"
// 	"path/filepath"
// 	"time"

// 	"github.com/PsychoPunkSage/ErgoFS/pkg/types"
// 	"github.com/PsychoPunkSage/ErgoFS/pkg/writer"
// )

// func main() {
// 	// Parse command line flags
// 	imgPath := flag.String("o", "", "Output image path")
// 	srcPath := flag.String("i", "", "Source directory or file path")
// 	blockSize := flag.Int("b", 4096, "Block size (default: 4096)")
// 	debugLevel := flag.Int("d", int(types.EROFS_INFO), "Debug level (0-4)")
// 	ignoreMtime := flag.Bool("ignore-mtime", false, "Ignore file modification times")
// 	timestamp := flag.Int64("T", 0, "Use specified UNIX timestamp for all files")
// 	uuid := flag.String("U", "", "Specify filesystem UUID (random if not provided)")
// 	label := flag.String("L", "", "Specify volume label")
// 	showVersion := flag.Bool("V", false, "Show version information")
// 	showHelp := flag.Bool("h", false, "Show help information")

// 	flag.Parse()

// 	// Set debug level
// 	types.SetDebugLevel(uint8(*debugLevel))

// 	// Show version and exit
// 	if *showVersion {
// 		fmt.Println("mkfs.erofs (erofs-go) version 1.0.0")
// 		os.Exit(0)
// 	}

// 	// Show help and exit
// 	if *showHelp {
// 		showUsage()
// 		os.Exit(0)
// 	}

// 	// Validate required parameters
// 	if *imgPath == "" {
// 		fmt.Fprintln(os.Stderr, "Error: Output image path (-o) is required")
// 		showUsage()
// 		os.Exit(1)
// 	}

// 	if *srcPath == "" {
// 		fmt.Fprintln(os.Stderr, "Error: Source directory (-i) is required")
// 		showUsage()
// 		os.Exit(1)
// 	}

// 	// Validate source path exists
// 	srcInfo, err := os.Stat(*srcPath)
// 	if err != nil {
// 		fmt.Fprintf(os.Stderr, "Error: Source path %s not found or inaccessible: %v\n", *srcPath, err)
// 		os.Exit(1)
// 	}

// 	if !srcInfo.IsDir() {
// 		fmt.Fprintf(os.Stderr, "Error: Source path %s is not a directory\n", *srcPath)
// 		os.Exit(1)
// 	}

// 	// Validate block size
// 	if *blockSize < 512 || *blockSize > int(types.EROFS_MAX_BLOCK_SIZE) {
// 		fmt.Fprintf(os.Stderr, "Error: Block size must be between 512 and %d\n", types.EROFS_MAX_BLOCK_SIZE)
// 		os.Exit(1)
// 	}

// 	// Check block size is a power of 2
// 	if *blockSize&(*blockSize0) != 0 {
// 		fmt.Fprintf(os.Stderr, "Error: Block size must be a power of 2\n")
// 		os.Exit(1)
// 	}

// 	types.Erofs_info("Creating EROFS filesystem")
// 	types.Erofs_info("Source dir: %s", *srcPath)
// 	types.Erofs_info("Output file: %s", *imgPath)
// 	types.Erofs_info("Block size: %d", *blockSize)

// 	// Create configuration
// 	config := types.DefaultConfig()
// 	config.ImagePath = *imgPath
// 	config.SourcePath = *srcPath
// 	config.DebugLevel = uint8(*debugLevel)
// 	config.IgnoreMtime = *ignoreMtime
// 	if *timestamp != 0 {
// 		config.UnixTimestamp = *timestamp
// 		config.TimeInherit = types.TIMESTAMP_FIXED
// 	}

// 	// Create builder
// 	builder := writer.NewBuilder(config)

// 	// Set block size
// 	blockSizeBits := 0
// 	for bs := *blockSize; bs > 1; bs >>= 1 {
// 		blockSizeBits++
// 	}
// 	builder.SuperBlock.BlksizeBits = uint8(blockSizeBits)
// 	types.Erofs_debug("Block size bits: %d", blockSizeBits)

// 	// Set feature flags
// 	builder.SuperBlock.FeatureCompat = types.EROFS_FEATURE_COMPAT_SB_CHKSUM | types.EROFS_FEATURE_COMPAT_MTIME
// 	builder.SuperBlock.FeatureIncompat = types.EROFS_FEATURE_INCOMPAT_ZERO_PADDING

// 	// Set UUID if provided
// 	if *uuid != "" {
// 		// Parse UUID string and set it
// 		// This needs an implementation for your UUID parsing
// 	}

// 	// Set volume label if provided
// 	if *label != "" && len(*label) <= 16 {
// 		copy(builder.SuperBlock.VolumeName[:], []byte(*label))
// 	}

// 	// Set build time
// 	if *timestamp != 0 {
// 		builder.SuperBlock.BuildTime = *timestamp
// 		builder.SuperBlock.BuildTimeNsec = 0
// 	} else {
// 		now := time.Now()
// 		builder.SuperBlock.BuildTime = now.Unix()
// 		builder.SuperBlock.BuildTimeNsec = uint32(now.Nanosecond())
// 	}

// 	// Open output file
// 	types.Erofs_info("Opening output file")
// 	err = builder.Open()
// 	if err != nil {
// 		fmt.Fprintf(os.Stderr, "Error opening output file: %v\n", err)
// 		os.Exit(1)
// 	}

// 	// Create root inode
// 	types.Erofs_info("Creating root directory inode")
// 	err = builder.CreateRoot()
// 	if err != nil {
// 		fmt.Fprintf(os.Stderr, "Error creating root directory: %v\n", err)
// 		os.Exit(1)
// 	}

// 	// Build filesystem from source path
// 	types.Erofs_info("Building filesystem from: %s", *srcPath)
// 	err = builder.BuildFromPath(*srcPath)
// 	if err != nil {
// 		fmt.Fprintf(os.Stderr, "Error building filesystem: %v\n", err)
// 		os.Exit(1)
// 	}

// 	// Write all inodes
// 	types.Erofs_info("Writing inodes to disk")
// 	err = builder.WriteAllInodes()
// 	if err != nil {
// 		fmt.Fprintf(os.Stderr, "Error writing inodes: %v\n", err)
// 		os.Exit(1)
// 	}

// 	// Close and finalize the filesystem
// 	types.Erofs_info("Finalizing and closing filesystem")
// 	err = builder.Close()
// 	if err != nil {
// 		fmt.Fprintf(os.Stderr, "Error closing filesystem: %v\n", err)
// 		os.Exit(1)
// 	}

// 	// Report summary information
// 	blocks := uint32(builder.CurrentPos >> blockSizeBits)
// 	if builder.CurrentPos%(1<<blockSizeBits) != 0 {
// 		blocks++
// 	}

// 	types.Erofs_info("------")
// 	types.Erofs_info("Filesystem UUID: %x-%x-%x-%x",
// 		builder.SuperBlock.UUID[:4],
// 		builder.SuperBlock.UUID[4:6],
// 		builder.SuperBlock.UUID[6:8],
// 		builder.SuperBlock.UUID[8:])
// 	types.Erofs_info("Filesystem total blocks: %d (of %d-byte blocks)",
// 		blocks, *blockSize)
// 	types.Erofs_info("Filesystem total inodes: %d", len(builder.InodeMap))
// 	types.Erofs_info("Build completed.")

// 	// Run hexdump on the first 256 bytes for verification
// 	if uint8(*debugLevel) >= types.EROFS_DEBUG {
// 		file, err := os.Open(*imgPath)
// 		if err == nil {
// 			defer file.Close()

// 			data := make([]byte, 256)
// 			n, _ := file.Read(data)
// 			if n > 0 {
// 				types.Erofs_debug("First %d bytes of the image file:", n)
// 				types.DumpHex(data[:n], "IMG")
// 			}
// 		}
// 	}

// 	fmt.Printf("Successfully created EROFS image: %s\n", *imgPath)
// }

// func showUsage() {
// 	progName := filepath.Base(os.Args[0])
// 	fmt.Printf("Usage: %s [OPTIONS] -o IMAGE_FILE -i SOURCE_DIR\n", progName)
// 	fmt.Printf("Create an EROFS filesystem image from SOURCE_DIR.\n\n")
// 	fmt.Printf("Options:\n")
// 	fmt.Printf("  -o IMAGE_FILE       Output image file path\n")
// 	fmt.Printf("  -i SOURCE_DIR       Source directory\n")
// 	fmt.Printf("  -b BLOCK_SIZE       Set block size (default: 4096)\n")
// 	fmt.Printf("  -d LEVEL            Set debug level (0-4, default: 2)\n")
// 	fmt.Printf("  -T TIMESTAMP        Set a fixed UNIX timestamp for all files\n")
// 	fmt.Printf("  -U UUID             Set a specific filesystem UUID\n")
// 	fmt.Printf("  -L LABEL            Set volume label (max 16 bytes)\n")
// 	fmt.Printf("  -V                  Show version information\n")
// 	fmt.Printf("  -h                  Show this help\n")
// 	fmt.Printf("  --ignore-mtime      Ignore file modification times\n")
// }

// cmd/mkfs/main.go

package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/PsychoPunkSage/ErgoFS/pkg/types"
	"github.com/PsychoPunkSage/ErgoFS/pkg/util"
)

func main() {
	// // Parse command line flags
	// debugLevel := flag.Int("d", 5, "Debug level (0-9)")
	// imagePath := flag.String("o", "", "Output image file path")
	// inputPath := flag.String("i", "", "Input directory path")
	// blockSize := flag.Int("b", 4096, "Block size in bytes (must be power of 2)")
	// volumeLabel := flag.String("L", "", "Volume label (max 15 bytes)")
	// flag.Parse()

	// // Validate arguments
	// if *imagePath == "" {
	// 	fmt.Println("Error: Output image file path is required (-o)")
	// 	flag.Usage()
	// 	os.Exit(1)
	// }

	// if *inputPath == "" {
	// 	fmt.Println("Error: Input directory path is required (-i)")
	// 	flag.Usage()
	// 	os.Exit(1)
	// }

	// // Set debug level
	// types.GCfg.DebugLevel = *debugLevel

	// // Calculate block size bits
	// blockSizeBits := 0
	// tmpSize := *blockSize
	// for tmpSize > 1 {
	// 	tmpSize >>= 1
	// 	blockSizeBits++
	// }

	// // Validate block size
	// if (1 << blockSizeBits) != *blockSize {
	// 	types.Error("Block size %d is not a power of 2", *blockSize)
	// 	os.Exit(1)
	// }

	// // Check if input directory exists
	// inputStat, err := os.Stat(*inputPath)
	// if err != nil {
	// 	types.Error("Cannot access input directory: %v", err)
	// 	os.Exit(1)
	// }

	// if !inputStat.IsDir() {
	// 	types.Error("Input path is not a directory: %s", *inputPath)
	// 	os.Exit(1)
	// }

	// // Initialize superblock info
	// sbi := types.NewSuperBlockInfo()
	// sbi.BlkSzBits = uint8(blockSizeBits)
	// sbi.DevName = *imagePath

	// // Set volume label if provided
	// if *volumeLabel != "" {
	// 	if len(*volumeLabel) > 15 {
	// 		types.Warning("Volume label too long, truncating to 15 bytes")
	// 		*volumeLabel = (*volumeLabel)[:15]
	// 	}
	// 	copy(sbi.VolumeName[:], []byte(*volumeLabel))
	// }

	// // Set current timestamp
	// now := time.Now()
	// sbi.BuildTime = uint64(now.Unix())
	// sbi.BuildTimeNsec = uint32(now.Nanosecond())

	// // Print filesystem creation information
	// types.Info("Creating EroFS filesystem on %s", *imagePath)
	// types.Info("Block size: %d bytes", *blockSize)
	// types.Info("Input directory: %s", *inputPath)

	// // Create or truncate the image file
	// file, err := os.Create(*imagePath)
	// if err != nil {
	// 	types.Error("Failed to create image file: %v", err)
	// 	os.Exit(1)
	// }
	// file.Close()

	// // Initialize buffer manager at block 0
	// sbi.Bmgr = util.InitBufferManager(sbi, 0)
	// if sbi.Bmgr == nil {
	// 	types.Error("Failed to initialize buffer manager")
	// 	os.Exit(1)
	// }

	// // Reserve space for superblock
	// sbBh, err := types.ReserveSuperblock(sbi.Bmgr)
	// // tempSb := &SuperBlock{BlkSzBits: sbi.BlkSzBits} // Create a temporary SuperBlock with necessary fields
	// if err != nil {
	// 	types.Error("Failed to reserve superblock: %v", err)
	// 	os.Exit(1)
	// }

	// // Create root inode
	// rootInode := types.NewInode(sbi)
	// rootInode.SetRoot()

	// // Set NID for the root inode
	// rootInode.Nid = 1
	// sbi.RootNid = 1 // Root directory inode number

	// // Walk the directory and collect files
	// fileCount := 0
	// err = filepath.Walk(*inputPath, func(path string, info os.FileInfo, err error) error {
	// 	if err != nil {
	// 		types.Warning("Error accessing path %s: %v", path, err)
	// 		return nil
	// 	}

	// 	fileCount++
	// 	return nil
	// })

	// if err != nil {
	// 	types.Error("Failed to walk input directory: %v", err)
	// 	os.Exit(1)
	// }

	// // Set inode count
	// sbi.Inos = uint64(fileCount)
	// types.Info("Found %d files/directories", fileCount)

	// // Set up metadata block address (right after superblock)
	// sbi.MetaBlkAddr = 1 // Block 0 is for superblock

	// // Set compatible features
	// sbi.SetFeature("sb_chksum")

	// // Write the superblock
	// var totalBlocks uint32
	// err = writer.WriteSuperblock(sbi, sbBh, &totalBlocks)
	// if err != nil {
	// 	types.Error("Failed to write superblock: %v", err)
	// 	os.Exit(1)
	// }

	// // Resize the device to the actual size used
	// err = util.DevResize(sbi, totalBlocks)
	// if err != nil {
	// 	types.Error("Failed to resize the device: %v", err)
	// 	os.Exit(1)
	// }

	// // Enable superblock checksum
	// if types.HasFeature(sbi, "sb_chksum") {
	// 	crc, err := writer.EnableSuperblockChecksum(sbi)
	// 	if err != nil {
	// 		types.Error("Failed to enable superblock checksum: %v", err)
	// 		os.Exit(1)
	// 	}
	// 	types.Info("Superblock checksum: 0x%08x", crc)
	// }

	// types.Info("EroFS filesystem creation complete")
	// types.Info("Total blocks: %d", totalBlocks)
	// types.Info("Total inodes: %d", sbi.Inos)

	// Define command-line flags
	dbgLevel := flag.Int("d", 0, "Debug level") // Default debug level = 0
	flag.Parse()
	// Get positional arguments
	args := flag.Args()
	if len(args) < 2 {
		fmt.Println("Usage: program -d <dbglevel> <image_path> <src_path>")
		os.Exit(1)
	}
	imagePath := args[0]
	srcPath := args[1]
	fmt.Printf("Debug Level: %d, Image Path: %s, Source Path: %s\n", *dbgLevel, imagePath, srcPath)

	types.GCfg.SourcePath = srcPath
	types.GCfg.ImagePath = imagePath
	types.GCfg.DebugLevel = *dbgLevel

	var err int
	var sbBh *types.BufferHead
	var nblocks uint32 = 0
	var crc uint32

	types.InitConfigure()
	types.MkfsDefaultOptions(&types.GSbi)

	if types.GSbi.BDev == nil {
		fmt.Println("I'm here")
		types.GSbi.BDev = &types.ErofsVFile{} // or appropriate initialization
	}

	// types.ShowProgs() // args??
	errs := util.DevOpen(&types.GSbi, types.GCfg.ImagePath, os.O_RDWR|0) // Assuming incremental_mode = true
	if errs != nil {
		fmt.Println("Something went wrong")
		return
	}

	// increamental mode = true
	types.GSbi.Bmgr = types.ErofsBufferInit(&types.GSbi, 0)
	if types.GSbi.Bmgr == nil {
		// types.Exit("failed to initialize buffer manager")
		err = -types.ENOMEM
	}

	sbBh, errors := types.ReserveSuperblock(types.GSbi.Bmgr)
	if errors != nil {
		fmt.Println("Failed to reserve superblock:", err)
		return // goto exit
	}

	fmt.Println("Superblock reserved successfully:", sbBh)

	types.UUIDGenerate(types.GSbi.UUID[:])

	err = types.ZErofsCompressInit(&types.GSbi, sbBh)
	if err != 0 {
		fmt.Println("Failed to initialize compressor")
		return // goto exit
	}

	// flush all buffers except for superblock
	err = types.ErofsBflush(types.GSbi.Bmgr, nil)
	if err != 0 {
		fmt.Println("Failed to flush buffers")
		return // goto exit
	}

	err = types.WriteSuperBlock(&types.GSbi, sbBh, &nblocks)
	if err != 0 {
		fmt.Println("Failed to write SB")
		return // goto exit
	}

	// flush all remaining buffers
	err = types.ErofsBflush(types.GSbi.Bmgr, nil)
	if err != 0 {
		fmt.Println("Failed to flush buffers")
		return // goto exit
	}

	err = types.ErofsDevResize(&types.GSbi, nblocks)

	if err == 0 && types.GSbi.ErofsSbHasSbChksum() {
		err = types.ErofsEnableSbChksum(&types.GSbi, &crc)
		if err == 0 {
			fmt.Printf("SuperBlock checksum 0x%08x written\n", crc)
			return // goto exit
		}
	}
}
