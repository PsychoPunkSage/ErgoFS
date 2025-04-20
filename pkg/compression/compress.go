package compression

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"math/bits"
	"os"
	"regexp"
	"strconv"
	"strings"
	"syscall"
	"unsafe"

	errs "github.com/PsychoPunkSage/ErgoFS/pkg/errors"
	"github.com/PsychoPunkSage/ErgoFS/pkg/types"
)

// var zErofsMtEnabled bool
// Global list head for compression hints
var CompressHintsHead types.ListHead

// ErofsCompressHints represents compression hints for specific files
type ErofsCompressHints struct {
	List                types.ListHead
	Reg                 *regexp.Regexp
	PhysicalClusterblks uint
	AlgorithmType       uint8
}

type ErofsAlgorithm struct {
	Name      string
	C         *ErofsCompressor
	ID        uint
	OptimiSor bool // its name won't be shown as a supported algorithm
}

type ErofsCompressor struct {
	DefaultLevel    int
	BestLevel       int
	DefaultDictSize uint32
	MaxDictSize     uint32

	Init        func(*ErofsCompress) int
	Exit        func(*ErofsCompress) int
	Reset       func(*ErofsCompress)
	SetLevel    func(*ErofsCompress, int) int
	SetDictSize func(*ErofsCompress, uint32) int

	CompressDestSize func(c *ErofsCompress, src []byte, srcSize *uint,
		dst []byte, dstSize uint) int
}

type ErofsCompress struct {
	Sbi               *types.SuperBlkInfo
	Alg               *ErofsAlgorithm
	CompressThreshold uint
	CompressionLevel  int
	DictSize          uint
	PrivateData       interface{}
}

type ErofsCompressCfg struct {
	Handle        ErofsCompress
	AlgorithmType uint
	Enable        bool
}

// ZErofsLzmaCfgs corresponds to the LZMA compression configuration (16 bytes total)
type ZErofsLzmaCfgs struct {
	DictSize uint32
	Format   uint16
	Reserved [8]byte
}

// ZErofsDeflateCfgs corresponds to the Deflate compression configuration (8 bytes total)
type ZErofsDeflateCfgs struct {
	WindowBits uint8
	Reserved   [5]byte
}

// ZErofsZstdCfgs corresponds to the ZSTD compression configuration (8 bytes total)
type ZErofsZstdCfgs struct {
	Format    uint8
	WindowLog uint8
	Reserved  [4]byte
}

// ErofsAlgs defines all supported compression algorithms
var erofsAlgs = []ErofsAlgorithm{
	{
		Name:      "lz4",
		C:         &ErofsCompressorLz4,
		ID:        types.Z_EROFS_COMPRESSION_LZ4,
		OptimiSor: false,
	},
	// {
	// 	Name:      "lz4hc",
	// 	C:         getCompressorLZ4HC(),
	// 	ID:        Z_EROFS_COMPRESSION_LZ4,
	// 	OptimiSor: true,
	// },
	// {
	// 	Name:      "lzma",
	// 	C:         getCompressorLZMA(),
	// 	ID:        Z_EROFS_COMPRESSION_LZMA,
	// 	OptimiSor: false,
	// },
	// {
	// 	Name:      "deflate",
	// 	C:         &ErofsCompressorDeflate,
	// 	ID:        Z_EROFS_COMPRESSION_DEFLATE,
	// 	OptimiSor: false,
	// },
	// {
	// 	Name:      "libdeflate",
	// 	C:         getCompressorLibDeflate(),
	// 	ID:        Z_EROFS_COMPRESSION_DEFLATE,
	// 	OptimiSor: true,
	// },
	// {
	// 	Name:      "zstd",
	// 	C:         getCompressorZSTD(),
	// 	ID:        Z_EROFS_COMPRESSION_ZSTD,
	// 	OptimiSor: false,
	// },
}

var ErofsCCfg [types.EROFS_MAX_COMPR_CFGS]ErofsCompressCfg

func ErofsLoadCompressHints(sbi *types.SuperBlkInfo) int {
	buf := make([]byte, types.PATH_MAX+100)
	line := uint(1)
	maxPclustersize := uint(0)
	ret := 0

	if types.GCfg.CompressHintsFile == "" {
		return 0
	}

	f, err := os.Open(types.GCfg.CompressHintsFile)
	if err != nil {
		return -errs.ENOENT
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		lineText := scanner.Text()

		// Skip comments and empty lines
		if len(lineText) == 0 || lineText[0] == '#' || lineText[0] == '\n' {
			line++
			continue
		}

		// Copy line to buffer (to match C strtok behavior)
		copy(buf, []byte(lineText))
		bufStr := string(buf[:len(lineText)])

		// Split the line (equivalent to strtok in C)
		fields := strings.Fields(bufStr)
		if len(fields) < 2 {
			fmt.Printf("cannot find a match pattern at line %d\n", line)
			ret = -errs.EINVAL
			goto out
		}

		// Parse pclustersize
		pclustersize, err := strconv.Atoi(fields[0])
		if err != nil {
			fmt.Printf("invalid pclustersize at line %d\n", line)
			ret = -int(syscall.EINVAL)
			goto out
		}

		var alg string
		var pattern string

		if len(fields) == 2 {
			// Only pattern is provided, no algorithm
			alg = ""
			pattern = fields[1]
		} else {
			// Both algorithm and pattern are provided
			alg = fields[1]
			pattern = fields[2]
		}

		if pattern == "" {
			fmt.Printf("cannot find a match pattern at line %d\n", line)
			ret = -int(syscall.EINVAL)
			goto out
		}

		var ccfg uint
		if alg == "" {
			ccfg = 0
		} else {
			ccfgVal, err := strconv.Atoi(alg)
			if err != nil || ccfgVal < 0 {
				fmt.Printf("invalid compressing configuration \"%s\" at line %d\n", alg, line)
				ret = -int(syscall.EINVAL)
				goto out
			}

			ccfg = uint(ccfgVal)
			if ccfg >= uint(types.EROFS_MAX_COMPR_CFGS) ||
				types.GCfg.CompressionOptions[ccfg].Algorithm == "" {
				fmt.Printf("invalid compressing configuration \"%s\" at line %d\n", alg, line)
				ret = -int(syscall.EINVAL)
				goto out
			}
		}

		if uint32(pclustersize)%types.ErofsBlkSiz(sbi) != 0 {
			fmt.Printf("invalid physical clustersize %d, use default pclusterblks %d\n",
				pclustersize, types.GCfg.MkfsPclusterSizeDef)
			line++
			continue
		}

		ErofsInsertCompressHints(pattern,
			uint(pclustersize/int(types.ErofsBlkSiz(sbi))),
			ccfg)

		if uint(pclustersize) > maxPclustersize {
			maxPclustersize = uint(pclustersize)
		}

		line++
	}

	if types.GCfg.MkfsPclusterSizeMax < uint32(maxPclustersize) {
		types.GCfg.MkfsPclusterSizeMax = uint32(maxPclustersize)
		fmt.Printf("update max pclustersize to %d\n", types.GCfg.MkfsPclusterSizeMax)
	}

out:
	// The defer will close the file
	return ret
}

// ErofsInsertCompressHints inserts a compression hint
func ErofsInsertCompressHints(s string, blks, algorithmType uint) int {
	ch := &ErofsCompressHints{
		PhysicalClusterblks: blks,
		AlgorithmType:       uint8(algorithmType),
	}

	// Initialize the list entry
	types.InitListHead(&ch.List)

	// Compile the regular expression
	reg, err := regexp.Compile(s)
	if err != nil {
		fmt.Printf("invalid regex %s (%s)\n", s, err.Error())
		return -errs.EINVAL
	}
	ch.Reg = reg

	// Add to the list
	types.ListAddTail(&ch.List, &CompressHintsHead)

	fmt.Printf("compress hint %s (%d) is inserted\n", s, blks)
	return 0
}

func ZErofsCompressInit(sbi *types.SuperBlkInfo, sbBh *types.BufferHead) int {
	maxDictSize := make([]uint32, types.Z_EROFS_COMPRESSION_MAX)
	availableComprAlgs := uint32(0)

	if len(types.GCfg.CompressionOptions) == 0 {
		fmt.Println("No compression options configured")
		return -1 // Or appropriate return code for no compression
	}

	// Make sure ErofsCCfg has the same length as GCfg.CompressionOptions
	if len(ErofsCCfg) < len(types.GCfg.CompressionOptions) {
		// Either resize ErofsCCfg or return an error
		fmt.Println("ErofsCCfg not properly initialized")
		return -1
	}

	for i := 0; i < len(types.GCfg.CompressionOptions); i++ {
		// Skip empty algorithms
		if types.GCfg.CompressionOptions[i].Algorithm == "" {
			continue
		}

		c := &ErofsCCfg[i].Handle

		ret := erofsCompressorInit(sbi, c, types.GCfg.CompressionOptions[i].Algorithm, types.GCfg.CompressionOptions[i].Level, types.GCfg.CompressionOptions[i].DictSize)
		if ret != 0 {
			return ret
		}

		id, err := zErofsGetCompressAlgorithmID(c)
		if err != nil {
			// return fmt.Errorf("failed to get compress algorithm ID: %w", err)
			return -1
		}

		ErofsCCfg[i].AlgorithmType = id
		ErofsCCfg[i].Enable = true
		availableComprAlgs |= 1 << ErofsCCfg[i].AlgorithmType
		if ErofsCCfg[i].AlgorithmType != types.Z_EROFS_COMPRESSION_LZ4 {
			types.ErofsSbSetComprCfgs(sbi)
			//ErofsSbSetComprCfgs()
		}
		if uint32(c.DictSize) > maxDictSize[id] {
			maxDictSize[id] = uint32(c.DictSize)
		}
	}

	// If primary algorithm is empty (compression off),
	// clear 0PADDING feature for old kernel compatibility
	if availableComprAlgs == 0 ||
		(types.GCfg.LegacyCompress && availableComprAlgs == 1) {
		types.ErofsSbClearLz40Padding(sbi)
	}

	if availableComprAlgs == 0 {
		return 0
	}

	if sbBh == nil {
		dalg := availableComprAlgs & ^uint32(sbi.AvailableComprAlgs)

		if dalg != 0 {
			// ErofsErr("unavailable algorithms 0x%x on incremental builds", dalg)
			return -errs.EOPNOTSUPP
		}
		if (availableComprAlgs&(1<<types.Z_EROFS_COMPRESSION_LZ4) != 0) &&
			(uint32(sbi.Lz4.MaxPclusterBlks)<<sbi.BlkSzBits < types.GCfg.MkfsPclusterSizeMax) {
			// ErofsErr("pclustersize %u is too large on incremental builds",
			// cfg.CMkfsPclustersizeMax)
			return -errs.EOPNOTSUPP
		}
	} else {
		sbi.AvailableComprAlgs = uint16(availableComprAlgs)
	}

	// If big pcluster is enabled, an extra CBLKCNT lcluster index needs
	// to be loaded to get those compressed block counts
	if types.GCfg.MkfsPclusterSizeMax > types.ErofsBlkSiz(sbi) {
		if types.GCfg.MkfsPclusterSizeMax > types.Z_EROFS_PCLUSTER_MAX_SIZE {
			// ErofsErr("unsupported pclustersize %u (too large)",
			// 	GCfg.MkfsPclustersizeMax)
			return -errs.EINVAL
		}
		types.ErofsSbSetBigPcluster(sbi)
	}
	if types.GCfg.MkfsPclusterSizePacked > types.GCfg.MkfsPclusterSizeMax {
		// ErofsErr("invalid pclustersize for the packed file %u",
		// 	GCfg.CMkfsPclusterSizePacked)
		return -errs.EINVAL
	}

	if sbBh != nil && types.ErofsSbHasComprCfgs(sbi) {
		ret := ZErofsBuildComprCfgs(sbi, sbBh, maxDictSize)
		if ret != 0 {
			return ret
		}
	}

	// zErofsMtEnabled = false
	// // Multi-threading support would go here
	// // In Go, we would use goroutines instead of pthreads
	// if GCfg.MtWorkers >= 1 {
	// 	// Check if features incompatible with multi-threading are enabled
	// 	if GCfg.Dedupe || (GCfg.Fragments && !GCfg.AllFragments) {
	// 		if GCfg.Dedupe {
	// 			// ErofsWarn("multi-threaded dedupe is NOT implemented for now")
	// 		}
	// 		if GCfg.Fragments {
	// 			// ErofsWarn("multi-threaded fragments is NOT implemented for now")
	// 		}
	// 		GCfg.MtWorkers = 0
	// 	}
	// }

	// if GCfg.MtWorkers >= 1 {
	// 	// Initialize worker pool
	// 	ret := ErofsAllocWorkqueue(&zErofsMtCtrl.Wq,
	// 		GCfg.MtWorkers,
	// 		GCfg.MtWorkers<<2,
	// 		ZErofsMtWqTlsAlloc,
	// 		ZErofsMtWqTlsFree)
	// 	if ret != 0 {
	// 		return ret
	// 	}
	// 	zErofsMtEnabled = true
	// }

	// // Initialize synchronization primitives
	// GIctx.mutex = sync.Mutex{}
	// GIctx.cond = sync.NewCond(&GIctx.mutex)

	return 0
}

// ZErofsBuildComprCfgs builds compression configurations
func ZErofsBuildComprCfgs(sbi *types.SuperBlkInfo, sbBh *types.BufferHead, maxDictSize []uint32) int {
	bh := sbBh
	ret := 0

	// Process LZ4 compression if available
	if sbi.AvailableComprAlgs&(1<<types.Z_EROFS_COMPRESSION_LZ4) != 0 {
		// Create LZ4 configuration structure
		type Lz4AlgConfig struct {
			Size uint16
			Lz4  ZErofsLz4Cfgs
		}

		lz4alg := Lz4AlgConfig{
			Size: uint16(unsafe.Sizeof(ZErofsLz4Cfgs{})),
			Lz4: ZErofsLz4Cfgs{
				MaxDistance:     sbi.Lz4.MaxDistance,
				MaxPclusterBlks: uint16(types.GCfg.MkfsPclusterSizeMax >> sbi.BlkSzBits),
			},
		}

		// Convert to little endian
		lz4algBytes := make([]byte, unsafe.Sizeof(lz4alg))
		binary.LittleEndian.PutUint16(lz4algBytes[0:2], lz4alg.Size)
		binary.LittleEndian.PutUint16(lz4algBytes[2:4], lz4alg.Lz4.MaxDistance)
		lz4algBytes[4] = byte(lz4alg.Lz4.MaxPclusterBlks)

		// Attach buffer
		bh, err := types.Battach(bh, types.META, uint32(len(lz4algBytes)))
		if err < 0 {
			// error.New()
			return err
		}

		// Map and write data
		types.MapBh(nil, bh.Block)
		ret = types.ErofsDevWrite(sbi, lz4algBytes, types.BhTell(bh, false), len(lz4algBytes))
		bh.Op = &types.DropDirectlyBhops
	}

	// Process LZMA compression if available
	HaveLibLZMA := false // PPS: REMOVE
	if HaveLibLZMA && sbi.AvailableComprAlgs&(1<<types.Z_EROFS_COMPRESSION_LZMA) != 0 {
		// Create LZMA configuration structure
		type LzmaAlgConfig struct {
			Size uint16
			Lzma ZErofsLzmaCfgs
		}

		lzmaalg := LzmaAlgConfig{
			Size: uint16(unsafe.Sizeof(ZErofsLzmaCfgs{})),
			Lzma: ZErofsLzmaCfgs{
				DictSize: maxDictSize[types.Z_EROFS_COMPRESSION_DEFLATE],
			},
		}

		// Convert to little endian
		lzmaalgBytes := make([]byte, unsafe.Sizeof(lzmaalg))
		binary.LittleEndian.PutUint16(lzmaalgBytes[0:2], lzmaalg.Size)
		binary.LittleEndian.PutUint32(lzmaalgBytes[2:6], lzmaalg.Lzma.DictSize)

		// Attach buffer
		bh, err := types.Battach(bh, types.META, uint32(len(lzmaalgBytes)))
		if err < 0 {
			return err
		}

		// Map and write data
		types.MapBh(nil, bh.Block)
		ret = types.ErofsDevWrite(sbi, lzmaalgBytes, types.BhTell(bh, false), len(lzmaalgBytes))
		bh.Op = &types.DropDirectlyBhops
	}

	// Process DEFLATE compression if available
	if sbi.AvailableComprAlgs&(1<<types.Z_EROFS_COMPRESSION_DEFLATE) != 0 {
		// Create DEFLATE configuration structure
		type DeflateAlgConfig struct {
			Size uint16
			Z    ZErofsDeflateCfgs
		}

		zalg := DeflateAlgConfig{
			Size: uint16(unsafe.Sizeof(ZErofsDeflateCfgs{})),
			Z: ZErofsDeflateCfgs{
				WindowBits: uint8(bits.TrailingZeros32(maxDictSize[types.Z_EROFS_COMPRESSION_DEFLATE])),
			},
		}

		// Convert to little endian
		zalgBytes := make([]byte, unsafe.Sizeof(zalg))
		binary.LittleEndian.PutUint16(zalgBytes[0:2], zalg.Size)
		binary.LittleEndian.PutUint32(zalgBytes[2:6], uint32(zalg.Z.WindowBits))

		// Attach buffer
		bh, err := types.Battach(bh, types.META, uint32(len(zalgBytes)))
		if err < 0 {
			return err
		}

		// Map and write data
		types.MapBh(nil, bh.Block)
		ret = types.ErofsDevWrite(sbi, zalgBytes, types.BhTell(bh, false), len(zalgBytes))
		bh.Op = &types.DropDirectlyBhops
	}

	// Process ZSTD compression if available
	HaveLibZSTD := false // PPS:: Remove
	if HaveLibZSTD && sbi.AvailableComprAlgs&(1<<types.Z_EROFS_COMPRESSION_ZSTD) != 0 {
		// Create ZSTD configuration structure
		type ZstdAlgConfig struct {
			Size uint16
			Z    ZErofsZstdCfgs
		}

		zalg := ZstdAlgConfig{
			Size: uint16(unsafe.Sizeof(ZErofsZstdCfgs{})),
			Z: ZErofsZstdCfgs{
				WindowLog: uint8(bits.TrailingZeros32(maxDictSize[types.Z_EROFS_COMPRESSION_ZSTD]) - 10),
			},
		}

		// Convert to little endian
		zalgBytes := make([]byte, unsafe.Sizeof(zalg))
		binary.LittleEndian.PutUint16(zalgBytes[0:2], zalg.Size)
		zalgBytes[2] = zalg.Z.WindowLog

		// Attach buffer
		bh, err := types.Battach(bh, types.META, uint32(len(zalgBytes)))
		if err < 0 {
			return err
		}

		// Map and write data
		types.MapBh(nil, bh.Block)
		ret = types.ErofsDevWrite(sbi, zalgBytes, types.BhTell(bh, false), len(zalgBytes))
		bh.Op = &types.DropDirectlyBhops
	}

	return ret
}

func zErofsGetCompressAlgorithmID(c *ErofsCompress) (uint, error) {
	if c == nil || c.Alg == nil {
		return 0, fmt.Errorf("invalid compressor: algorithm is nil")
	}
	return c.Alg.ID, nil
}

// ErofsCompressorInit initializes a compressor
func erofsCompressorInit(sbi *types.SuperBlkInfo, c *ErofsCompress,
	algName string, compressionLevel int, dictSize uint32) int {
	c.Sbi = sbi

	// Should be written in "minimum compression ratio * 100"
	c.CompressThreshold = 100
	c.CompressionLevel = -1
	c.DictSize = 0

	if algName == "" {
		c.Alg = nil
		return 0
	}

	ret := -errs.EINVAL
	for i := 0; i < len(erofsAlgs); i++ {
		if algName != "" && algName != erofsAlgs[i].Name {
			continue
		}

		if erofsAlgs[i].C == nil {
			continue
		}

		if erofsAlgs[i].C.SetLevel != nil {
			ret = erofsAlgs[i].C.SetLevel(c, compressionLevel)
			if ret != 0 {
				// errors.New(fmt.Sprintf("failed to set compression level %d for %s",
				// 	compressionLevel, algName))
				return ret
			}
		} else if compressionLevel >= 0 {
			// errors.New(fmt.Sprintf("compression level %d is not supported for %s",
			// 	compressionLevel, algName))
			return -errs.EINVAL
		}

		if erofsAlgs[i].C.SetDictSize != nil {
			ret = erofsAlgs[i].C.SetDictSize(c, dictSize)
			if ret != 0 {
				// fmt.Errorf("failed to set dict size %d for %s", dictSize, algName)
				return ret
			}
		} else if dictSize > 0 {
			// errors.New(fmt.Sprintf("dict size is not supported for %s",
			// 	algName))
			return -errs.EINVAL
		}

		ret = erofsAlgs[i].C.Init(c)
		if ret != 0 {
			return ret
		}

		// If we get here, init succeeded
		c.Alg = &erofsAlgs[i]
		return 0
	}

	// errors.New(fmt.Sprintf("Cannot find a valid compressor %s", algName))
	return ret
}
