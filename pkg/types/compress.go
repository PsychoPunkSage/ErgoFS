package types

import (
	"errors"
	"fmt"
)

// import "cosmossdk.io/errors"

// "github.com/PsychoPunkSage/ErgoFS/pkg/types/compressor"

var zErofsMtEnabled bool

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
				errors.New(fmt.Sprintf("failed to set compression level %d for %s",
					compressionLevel, algName))
				return ret
			}
		} else if compressionLevel >= 0 {
			errors.New(fmt.Sprintf("compression level %d is not supported for %s",
				compressionLevel, algName))
			return -EINVAL
		}

		if erofsAlgs[i].C.SetDictSize != nil {
			ret = erofsAlgs[i].C.SetDictSize(c, dictSize)
			if ret != 0 {
				errors.New(fmt.Sprintf("failed to set dict size %u for %s",
					dictSize, algName))
				return ret
			}
		} else if dictSize > 0 {
			errors.New(fmt.Sprintf("dict size is not supported for %s",
				algName))
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

	errors.New(fmt.Sprintf("Cannot find a valid compressor %s", algName))
	return ret
}
