package types

import "regexp"

// struct erofs_compress_hints {
// 	struct list_head list;

// 	regex_t reg;
// 	unsigned int physical_clusterblks;
// 	unsigned char algorithmtype;
// };

type ErofsCompressHints struct {
	List                ListHead
	Reg                 *regexp.Regexp
	physicalClusterblks uint
	algorithmType       uint
}

func zErodsApplyCompressHints(inode *ErofsInode) bool {
	var r *ErofsCompressHints
	var pclusterblks, algorithmtype uint

	if inode.ZPhysicalClusterblks != 0 {
		return true
	}

	s := ErofsFspath(inode.ISrcpath)
	pclusterblks = uint(GCfg.MkfsPclusterSizeDef) >> uint(inode.Sbi.BlkSzBits)
	algorithmtype = 0

	// ListForEachEntry()
	// Create a function to process each compression hint
	processHint := func(item interface{}) bool {
		r := item.(*ErofsCompressHints)

		// Match RegEx pattern
		if r.Reg.MatchString(s) {
			pclusterblks = r.physicalClusterblks
			algorithmtype = r.algorithmType
			return false
		}

		return true
	}

	// Iterate through all compression hints
	ListForEachEntry(&r.List, &ErofsCompressHints{}, "List", processHint)

	inode.ZPhysicalClusterblks = uint8(pclusterblks)
	inode.ZAlgorithmType[0] = uint8(algorithmtype)

	// pclusterblks is 0 means this file shouldn't be compressed
	return pclusterblks != 0
}
