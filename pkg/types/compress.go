package types

import (
	"encoding/binary"
	"fmt"
	"math/bits"
	"unsafe"
)

// import "cosmossdk.io/errors"

// "github.com/PsychoPunkSage/ErgoFS/pkg/types/compressor"

// var zErofsMtEnabled bool

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
	Sbi               *SuperBlkInfo
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

type ZErofsLz4Cfgs struct {
	MaxDistance     uint16
	MaxPclusterBlks uint16
	Reserved        [10]byte
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

var ErofsCCfg [EROFS_MAX_COMPR_CFGS]ErofsCompressCfg

func ZErofsCompressInit(sbi *SuperBlkInfo, sbBh *BufferHead) int {
	maxDictSize := make([]uint32, Z_EROFS_COMPRESSION_MAX)
	availableComprAlgs := uint32(0)

	for i := 0; GCfg.CompressionOptions[i].Algorithm != ""; i++ {
		c := &ErofsCCfg[i].Handle

		ret := erofsCompressorInit(sbi, c, GCfg.CompressionOptions[i].Algorithm, GCfg.CompressionOptions[i].Level, GCfg.CompressionOptions[i].DictSize)
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
		if ErofsCCfg[i].AlgorithmType != Z_EROFS_COMPRESSION_LZ4 {
			sbi.ErofsSbSetComprCfgs()
			//ErofsSbSetComprCfgs()
		}
		if uint32(c.DictSize) > maxDictSize[id] {
			maxDictSize[id] = uint32(c.DictSize)
		}
	}

	// If primary algorithm is empty (compression off),
	// clear 0PADDING feature for old kernel compatibility
	if availableComprAlgs == 0 ||
		(GCfg.LegacyCompress && availableComprAlgs == 1) {
		sbi.ErofsSbClearLz40Padding()
	}

	if availableComprAlgs == 0 {
		return 0
	}

	if sbBh == nil {
		dalg := availableComprAlgs & ^uint32(sbi.AvailableComprAlgs)

		if dalg != 0 {
			// ErofsErr("unavailable algorithms 0x%x on incremental builds", dalg)
			return -EOPNOTSUPP
		}
		if (availableComprAlgs&(1<<Z_EROFS_COMPRESSION_LZ4) != 0) &&
			(uint32(sbi.Lz4.MaxPclusterBlks)<<sbi.BlkSzBits < GCfg.MkfsPclusterSizeMax) {
			// ErofsErr("pclustersize %u is too large on incremental builds",
			// cfg.CMkfsPclustersizeMax)
			return -EOPNOTSUPP
		}
	} else {
		sbi.AvailableComprAlgs = uint16(availableComprAlgs)
	}

	// If big pcluster is enabled, an extra CBLKCNT lcluster index needs
	// to be loaded to get those compressed block counts
	if GCfg.MkfsPclusterSizeMax > ErofsBlkSiz(sbi) {
		if GCfg.MkfsPclusterSizeMax > Z_EROFS_PCLUSTER_MAX_SIZE {
			// ErofsErr("unsupported pclustersize %u (too large)",
			// 	GCfg.MkfsPclustersizeMax)
			return -EINVAL
		}
		sbi.ErofsSbSetBigPcluster()
	}
	if GCfg.MkfsPclusterSizePacked > GCfg.MkfsPclusterSizeMax {
		// ErofsErr("invalid pclustersize for the packed file %u",
		// 	GCfg.CMkfsPclusterSizePacked)
		return -EINVAL
	}

	if sbBh != nil && sbi.ErofsSbHasComprCfgs() {
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
	// gIctx.mutex = sync.Mutex{}
	// gIctx.cond = sync.NewCond(&gIctx.mutex)

	return 0
}

// ZErofsBuildComprCfgs builds compression configurations
func ZErofsBuildComprCfgs(sbi *SuperBlkInfo, sbBh *BufferHead, maxDictSize []uint32) int {
	bh := sbBh
	ret := 0

	// Process LZ4 compression if available
	if sbi.AvailableComprAlgs&(1<<Z_EROFS_COMPRESSION_LZ4) != 0 {
		// Create LZ4 configuration structure
		type Lz4AlgConfig struct {
			Size uint16
			Lz4  ZErofsLz4Cfgs
		}

		lz4alg := Lz4AlgConfig{
			Size: uint16(unsafe.Sizeof(ZErofsLz4Cfgs{})),
			Lz4: ZErofsLz4Cfgs{
				MaxDistance:     sbi.Lz4.MaxDistance,
				MaxPclusterBlks: uint16(GCfg.MkfsPclusterSizeMax >> sbi.BlkSzBits),
			},
		}

		// Convert to little endian
		lz4algBytes := make([]byte, unsafe.Sizeof(lz4alg))
		binary.LittleEndian.PutUint16(lz4algBytes[0:2], lz4alg.Size)
		binary.LittleEndian.PutUint16(lz4algBytes[2:4], lz4alg.Lz4.MaxDistance)
		lz4algBytes[4] = byte(lz4alg.Lz4.MaxPclusterBlks)

		// Attach buffer
		bh, err := Battach(bh, META, uint32(len(lz4algBytes)))
		if err < 0 {
			// error.New()
			return err
		}

		// Map and write data
		MapBh(nil, bh.Block)
		ret = ErofsDevWrite(sbi, lz4algBytes, BhTell(bh, false), len(lz4algBytes))
		bh.Op = &DropDirectlyBhops
	}

	// Process LZMA compression if available
	HaveLibLZMA := false // PPS: REMOVE
	if HaveLibLZMA && sbi.AvailableComprAlgs&(1<<Z_EROFS_COMPRESSION_LZMA) != 0 {
		// Create LZMA configuration structure
		type LzmaAlgConfig struct {
			Size uint16
			Lzma ZErofsLzmaCfgs
		}

		lzmaalg := LzmaAlgConfig{
			Size: uint16(unsafe.Sizeof(ZErofsLzmaCfgs{})),
			Lzma: ZErofsLzmaCfgs{
				DictSize: maxDictSize[Z_EROFS_COMPRESSION_DEFLATE],
			},
		}

		// Convert to little endian
		lzmaalgBytes := make([]byte, unsafe.Sizeof(lzmaalg))
		binary.LittleEndian.PutUint16(lzmaalgBytes[0:2], lzmaalg.Size)
		binary.LittleEndian.PutUint32(lzmaalgBytes[2:6], lzmaalg.Lzma.DictSize)

		// Attach buffer
		bh, err := Battach(bh, META, uint32(len(lzmaalgBytes)))
		if err < 0 {
			return err
		}

		// Map and write data
		MapBh(nil, bh.Block)
		ret = ErofsDevWrite(sbi, lzmaalgBytes, BhTell(bh, false), len(lzmaalgBytes))
		bh.Op = &DropDirectlyBhops
	}

	// Process DEFLATE compression if available
	if sbi.AvailableComprAlgs&(1<<Z_EROFS_COMPRESSION_DEFLATE) != 0 {
		// Create DEFLATE configuration structure
		type DeflateAlgConfig struct {
			Size uint16
			Z    ZErofsDeflateCfgs
		}

		zalg := DeflateAlgConfig{
			Size: uint16(unsafe.Sizeof(ZErofsDeflateCfgs{})),
			Z: ZErofsDeflateCfgs{
				WindowBits: uint8(bits.TrailingZeros32(maxDictSize[Z_EROFS_COMPRESSION_DEFLATE])),
			},
		}

		// Convert to little endian
		zalgBytes := make([]byte, unsafe.Sizeof(zalg))
		binary.LittleEndian.PutUint16(zalgBytes[0:2], zalg.Size)
		binary.LittleEndian.PutUint32(zalgBytes[2:6], uint32(zalg.Z.WindowBits))

		// Attach buffer
		bh, err := Battach(bh, META, uint32(len(zalgBytes)))
		if err < 0 {
			return err
		}

		// Map and write data
		MapBh(nil, bh.Block)
		ret = ErofsDevWrite(sbi, zalgBytes, BhTell(bh, false), len(zalgBytes))
		bh.Op = &DropDirectlyBhops
	}

	// Process ZSTD compression if available
	HaveLibZSTD := false // PPS:: Remove
	if HaveLibZSTD && sbi.AvailableComprAlgs&(1<<Z_EROFS_COMPRESSION_ZSTD) != 0 {
		// Create ZSTD configuration structure
		type ZstdAlgConfig struct {
			Size uint16
			Z    ZErofsZstdCfgs
		}

		zalg := ZstdAlgConfig{
			Size: uint16(unsafe.Sizeof(ZErofsZstdCfgs{})),
			Z: ZErofsZstdCfgs{
				WindowLog: uint8(bits.TrailingZeros32(maxDictSize[Z_EROFS_COMPRESSION_ZSTD]) - 10),
			},
		}

		// Convert to little endian
		zalgBytes := make([]byte, unsafe.Sizeof(zalg))
		binary.LittleEndian.PutUint16(zalgBytes[0:2], zalg.Size)
		zalgBytes[2] = zalg.Z.WindowLog

		// Attach buffer
		bh, err := Battach(bh, META, uint32(len(zalgBytes)))
		if err < 0 {
			return err
		}

		// Map and write data
		MapBh(nil, bh.Block)
		ret = ErofsDevWrite(sbi, zalgBytes, BhTell(bh, false), len(zalgBytes))
		bh.Op = &DropDirectlyBhops
	}

	return ret
}

func zErofsGetCompressAlgorithmID(c *ErofsCompress) (uint, error) {
	if c == nil || c.Alg == nil {
		return 0, fmt.Errorf("invalid compressor: algorithm is nil")
	}
	return c.Alg.ID, nil
}

// ErofsAlgs defines all supported compression algorithms
var erofsAlgs = []ErofsAlgorithm{
	{
		Name:      "lz4",
		C:         &ErofsCompressorLz4,
		ID:        Z_EROFS_COMPRESSION_LZ4,
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

// ErofsCompressorInit initializes a compressor
func erofsCompressorInit(sbi *SuperBlkInfo, c *ErofsCompress,
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

	ret := -EINVAL
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
			return -EINVAL
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
			return -EINVAL
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
