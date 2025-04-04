package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/PsychoPunkSage/ErgoFS/pkg/compression"
	errs "github.com/PsychoPunkSage/ErgoFS/pkg/errors"
	"github.com/PsychoPunkSage/ErgoFS/pkg/types"
	"github.com/PsychoPunkSage/ErgoFS/pkg/util"
)

func main() {
	// Define command-line flags
	dbgLevel := flag.Int("d", 0, "Debug level") // Default debug level = 0
	compressHints := flag.String("C", "", "Path to compression hints file")
	compressionAlg := flag.String("c", "lz4", "Compression algorithm (lz4, lzma, etc.)")
	compressionLevel := flag.Int("l", -1, "Compression level")
	flag.Parse()

	// Get positional arguments
	args := flag.Args()
	if len(args) < 2 {
		fmt.Println("Usage: program [-d dbglevel] [-C compression_hints_file] [-c compression_alg] [-l compression_level] <image_path> <src_path>")
		os.Exit(1)
	}

	imagePath := args[0]
	srcPath := args[1]
	fmt.Printf("Debug Level: %d, Image Path: %s, Source Path: %s\n", *dbgLevel, imagePath, srcPath)

	types.GCfg.SourcePath = srcPath
	types.GCfg.ImagePath = imagePath
	types.GCfg.DebugLevel = *dbgLevel
	types.GCfg.CompressHintsFile = *compressHints

	var err int
	var sbBh *types.BufferHead
	var nblocks uint32 = 0
	var crc uint32

	// Initialize compression options if not already set
	if len(types.GCfg.CompressionOptions) == 0 {
		types.GCfg.CompressionOptions = []types.CompressionOption{
			{
				Algorithm: *compressionAlg,
				Level:     *compressionLevel,
				DictSize:  0,
			},
		}

		// Initialize the corresponding compression configurations
		tempCfg := make([]compression.ErofsCompressCfg, len(types.GCfg.CompressionOptions))
		copy(compression.ErofsCCfg[:], tempCfg)
	}

	types.InitConfigure()
	types.MkfsDefaultOptions(&types.GSbi)

	if types.GSbi.BDev == nil {
		fmt.Println("I'm here")
		types.GSbi.BDev = &types.ErofsVFile{} // or appropriate initialization
	}

	// types.ShowProgs() // args??
	errr := util.DevOpen(&types.GSbi, types.GCfg.ImagePath, os.O_RDWR|0) // Assuming incremental_mode = true
	// errs := util.DevOpen(&types.GSbi, types.GCfg.ImagePath, os.O_RDWR|os.O_TRUNC) // Assuming incremental_mode = true
	if errr != nil {
		fmt.Println("Something went wrong")
		return
	}

	// increamental mode = true
	types.GSbi.Bmgr = types.ErofsBufferInit(&types.GSbi, 0)
	if types.GSbi.Bmgr == nil {
		// types.Exit("failed to initialize buffer manager")
		err = -errs.ENOMEM
	}

	sbBh, errors := types.ReserveSuperblock(types.GSbi.Bmgr)
	if errors != nil {
		fmt.Println("Failed to reserve superblock:", err)
		return // goto exit
	}

	fmt.Println("Superblock reserved successfully:", sbBh)

	types.UUIDGenerate(types.GSbi.UUID[:])

	fmt.Printf("UUID generated successfully: %+v\n", types.GSbi.UUID)

	err = compression.ErofsLoadCompressHints(&types.GSbi)
	if err != 0 {
		fmt.Println("Failed to load compress hints")
		return // goto exit
	}

	err = compression.ZErofsCompressInit(&types.GSbi, sbBh)
	if err != 0 {
		fmt.Println("Failed to initialize compressor")
		return // goto exit
	}

	fmt.Println("Compress Initialization successfully Done")

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

	fmt.Println("Superblock successfully Written")

	// flush all remaining buffers
	// err = types.ErofsBflush(types.GSbi.Bmgr, nil)
	// if err != 0 {
	// 	fmt.Println("Failed to flush buffers")
	// 	return // goto exit
	// }

	err = types.ErofsDevResize(&types.GSbi, nblocks)

	if err == 0 && types.ErofsSbHasSbChksum(&types.GSbi) {
		err = types.ErofsEnableSbChksum(&types.GSbi, &crc)
		if err == 0 {
			fmt.Printf("SuperBlock checksum 0x%08x written\n", crc)
			return // goto exit
		}
	}
}
