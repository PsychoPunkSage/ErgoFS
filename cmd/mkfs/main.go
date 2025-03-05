package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/PsychoPunkSage/ErgoFS/pkg/types"
	"github.com/PsychoPunkSage/ErgoFS/pkg/writer"
)

func main() {
	// Parse command line flags
	imgPath := flag.String("o", "", "Output image path")
	srcPath := flag.String("i", "", "Source directory or file path")
	blockSize := flag.Int("b", 4096, "Block size (default: 4096)")
	debugLevel := flag.Int("d", int(types.EROFS_INFO), "Debug level (0-4)")
	ignoreMtime := flag.Bool("ignore-mtime", false, "Ignore file modification times")
	showVersion := flag.Bool("V", false, "Show version information")
	showHelp := flag.Bool("h", false, "Show help information")

	flag.Parse()

	// Set debug level
	types.SetDebugLevel(uint8(*debugLevel))

	// Show version and exit
	if *showVersion {
		fmt.Println("mkfs.erofs (erofs-go) version 1.0.0")
		os.Exit(0)
	}

	// Show help and exit
	if *showHelp {
		showUsage()
		os.Exit(0)
	}

	// Validate required parameters
	if *imgPath == "" {
		fmt.Fprintln(os.Stderr, "Error: Output image path (-o) is required")
		showUsage()
		os.Exit(1)
	}

	if *srcPath == "" {
		fmt.Fprintln(os.Stderr, "Error: Source directory (-i) is required")
		showUsage()
		os.Exit(1)
	}

	// Validate source path exists
	srcInfo, err := os.Stat(*srcPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: Source path %s not found or inaccessible: %v\n", *srcPath, err)
		os.Exit(1)
	}

	if !srcInfo.IsDir() {
		fmt.Fprintf(os.Stderr, "Error: Source path %s is not a directory\n", *srcPath)
		os.Exit(1)
	}

	// Validate block size
	if *blockSize < 512 || *blockSize > int(types.EROFS_MAX_BLOCK_SIZE) {
		fmt.Fprintf(os.Stderr, "Error: Block size must be between 512 and %d\n", types.EROFS_MAX_BLOCK_SIZE)
		os.Exit(1)
	}

	// Check block size is a power of 2
	if *blockSize&(*blockSize-1) != 0 {
		fmt.Fprintf(os.Stderr, "Error: Block size must be a power of 2\n")
		os.Exit(1)
	}

	types.Erofs_info("Creating EROFS filesystem")
	types.Erofs_info("Source dir: %s", *srcPath)
	types.Erofs_info("Output file: %s", *imgPath)
	types.Erofs_info("Block size: %d", *blockSize)

	// Create configuration
	config := types.DefaultConfig()
	config.ImagePath = *imgPath
	config.SourcePath = *srcPath
	config.DebugLevel = uint8(*debugLevel)
	config.IgnoreMtime = *ignoreMtime

	// Create builder
	builder := writer.NewBuilder(config)

	// Set block size
	blockSizeBits := 0
	for bs := *blockSize; bs > 1; bs >>= 1 {
		blockSizeBits++
	}
	builder.SuperBlock.BlksizeBits = uint8(blockSizeBits)
	types.Erofs_debug("Block size bits: %d", blockSizeBits)

	// Open output file
	types.Erofs_info("Opening output file")
	err = builder.Open()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error opening output file: %v\n", err)
		os.Exit(1)
	}

	// Create root inode
	types.Erofs_info("Creating root directory inode")
	err = builder.CreateRoot()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating root directory: %v\n", err)
		os.Exit(1)
	}

	// Build filesystem from source path
	types.Erofs_info("Building filesystem from: %s", *srcPath)
	err = builder.BuildFromPath(*srcPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error building filesystem: %v\n", err)
		os.Exit(1)
	}

	// Write all inodes
	types.Erofs_info("Writing inodes to disk")
	err = builder.WriteAllInodes()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error writing inodes: %v\n", err)
		os.Exit(1)
	}

	// Close and finalize the filesystem
	types.Erofs_info("Finalizing and closing filesystem")
	err = builder.Close()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error closing filesystem: %v\n", err)
		os.Exit(1)
	}

	// Run hexdump on the first 256 bytes for verification
	if uint8(*debugLevel) >= types.EROFS_DEBUG {
		file, err := os.Open(*imgPath)
		if err == nil {
			defer file.Close()

			data := make([]byte, 256)
			n, _ := file.Read(data)
			if n > 0 {
				types.Erofs_debug("First %d bytes of the image file:", n)
				types.DumpHex(data[:n], "IMG")
			}
		}
	}

	fmt.Printf("Successfully created EROFS image: %s\n", *imgPath)
}

func showUsage() {
	progName := filepath.Base(os.Args[0])
	fmt.Printf("Usage: %s [OPTIONS] -o IMAGE_FILE -i SOURCE_DIR\n", progName)
	fmt.Printf("Create an EROFS filesystem image from SOURCE_DIR.\n\n")
	fmt.Printf("Options:\n")
	fmt.Printf("  -o IMAGE_FILE       Output image file path\n")
	fmt.Printf("  -i SOURCE_DIR       Source directory\n")
	fmt.Printf("  -b BLOCK_SIZE       Set block size (default: 4096)\n")
	fmt.Printf("  -d LEVEL            Set debug level (0-4, default: 2)\n")
	fmt.Printf("  -V                   Show version information\n")
	fmt.Printf("  -h                   Show this help\n")
	fmt.Printf("  --ignore-mtime      Ignore file modification times\n")
}
