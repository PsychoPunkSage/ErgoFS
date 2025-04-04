package types

// lz4_0padding
func ErofsSbHasLz40Padding(sbi *SuperBlkInfo) bool {
	return sbi.FeatureIncompat&EROFS_FEATURE_INCOMPAT_ZERO_PADDING != 0
}
func ErofsSbSetLz40Padding(sbi *SuperBlkInfo) {
	sbi.FeatureIncompat |= EROFS_FEATURE_INCOMPAT_ZERO_PADDING
}
func ErofsSbClearLz40Padding(sbi *SuperBlkInfo) {
	sbi.FeatureIncompat &^= EROFS_FEATURE_INCOMPAT_ZERO_PADDING
}

// compr_cfgs
func ErofsSbHasComprCfgs(sbi *SuperBlkInfo) bool {
	return sbi.FeatureIncompat&EROFS_FEATURE_INCOMPAT_COMPR_CFGS != 0
}
func ErofsSbSetComprCfgs(sbi *SuperBlkInfo) {
	sbi.FeatureIncompat |= EROFS_FEATURE_INCOMPAT_COMPR_CFGS
}
func ErofsSbClearComprCfgs(sbi *SuperBlkInfo) {
	sbi.FeatureIncompat &^= EROFS_FEATURE_INCOMPAT_COMPR_CFGS
}

// big_pcluster
func ErofsSbHasBigPcluster(sbi *SuperBlkInfo) bool {
	return sbi.FeatureIncompat&EROFS_FEATURE_INCOMPAT_BIG_PCLUSTER != 0
}
func ErofsSbSetBigPcluster(sbi *SuperBlkInfo) {
	sbi.FeatureIncompat |= EROFS_FEATURE_INCOMPAT_BIG_PCLUSTER
}
func ErofsSbClearBigPcluster(sbi *SuperBlkInfo) {
	sbi.FeatureIncompat &^= EROFS_FEATURE_INCOMPAT_BIG_PCLUSTER
}

// chunked_file
func ErofsSbHasChunkedFile(sbi *SuperBlkInfo) bool {
	return sbi.FeatureIncompat&EROFS_FEATURE_INCOMPAT_CHUNKED_FILE != 0
}
func ErofsSbSetChunkedFile(sbi *SuperBlkInfo) {
	sbi.FeatureIncompat |= EROFS_FEATURE_INCOMPAT_CHUNKED_FILE
}
func ErofsSbClearChunkedFile(sbi *SuperBlkInfo) {
	sbi.FeatureIncompat &^= EROFS_FEATURE_INCOMPAT_CHUNKED_FILE
}

// device_table
func ErofsSbHasDeviceTable(sbi *SuperBlkInfo) bool {
	return sbi.FeatureIncompat&EROFS_FEATURE_INCOMPAT_DEVICE_TABLE != 0
}
func ErofsSbSetDeviceTable(sbi *SuperBlkInfo) {
	sbi.FeatureIncompat |= EROFS_FEATURE_INCOMPAT_DEVICE_TABLE
}
func ErofsSbClearDeviceTable(sbi *SuperBlkInfo) {
	sbi.FeatureIncompat &^= EROFS_FEATURE_INCOMPAT_DEVICE_TABLE
}

// ztailpacking
func ErofsSbHasZtailpacking(sbi *SuperBlkInfo) bool {
	return sbi.FeatureIncompat&EROFS_FEATURE_INCOMPAT_ZTAILPACKING != 0
}
func ErofsSbSetZtailpacking(sbi *SuperBlkInfo) {
	sbi.FeatureIncompat |= EROFS_FEATURE_INCOMPAT_ZTAILPACKING
}
func ErofsSbClearZtailpacking(sbi *SuperBlkInfo) {
	sbi.FeatureIncompat &^= EROFS_FEATURE_INCOMPAT_ZTAILPACKING
}

// fragments
func ErofsSbHasFragments(sbi *SuperBlkInfo) bool {
	return sbi.FeatureIncompat&EROFS_FEATURE_INCOMPAT_FRAGMENTS != 0
}
func ErofsSbSetFragments(sbi *SuperBlkInfo) {
	sbi.FeatureIncompat |= EROFS_FEATURE_INCOMPAT_FRAGMENTS
}
func ErofsSbClearFragments(sbi *SuperBlkInfo) {
	sbi.FeatureIncompat &^= EROFS_FEATURE_INCOMPAT_FRAGMENTS
}

// dedupe
func ErofsSbHasDedupe(sbi *SuperBlkInfo) bool {
	return sbi.FeatureIncompat&EROFS_FEATURE_INCOMPAT_DEDUPE != 0
}
func ErofsSbSetDedupe(sbi *SuperBlkInfo) {
	sbi.FeatureIncompat |= EROFS_FEATURE_INCOMPAT_DEDUPE
}
func ErofsSbClearDedupe(sbi *SuperBlkInfo) {
	sbi.FeatureIncompat &^= EROFS_FEATURE_INCOMPAT_DEDUPE
}

// xattr_prefixes
func ErofsSbHasXattrPrefixes(sbi *SuperBlkInfo) bool {
	return sbi.FeatureIncompat&EROFS_FEATURE_INCOMPAT_XATTR_PREFIXES != 0
}
func ErofsSbSetXattrPrefixes(sbi *SuperBlkInfo) {
	sbi.FeatureIncompat |= EROFS_FEATURE_INCOMPAT_XATTR_PREFIXES
}
func ErofsSbClearXattrPrefixes(sbi *SuperBlkInfo) {
	sbi.FeatureIncompat &^= EROFS_FEATURE_INCOMPAT_XATTR_PREFIXES
}

// sb_chksum
func ErofsSbHasSbChksum(sbi *SuperBlkInfo) bool {
	return sbi.FeatureCompat&EROFS_FEATURE_COMPAT_SB_CHKSUM != 0
}
func ErofsSbSetSbChksum(sbi *SuperBlkInfo) {
	sbi.FeatureCompat |= EROFS_FEATURE_COMPAT_SB_CHKSUM
}
func ErofsSbClearSbChksum(sbi *SuperBlkInfo) {
	sbi.FeatureCompat &^= EROFS_FEATURE_COMPAT_SB_CHKSUM
}

// xattr_filter
func ErofsSbHasXattrFilter(sbi *SuperBlkInfo) bool {
	return sbi.FeatureCompat&EROFS_FEATURE_COMPAT_XATTR_FILTER != 0
}
func ErofsSbSetXattrFilter(sbi *SuperBlkInfo) {
	sbi.FeatureCompat |= EROFS_FEATURE_COMPAT_XATTR_FILTER
}
func ErofsSbClearXattrFilter(sbi *SuperBlkInfo) {
	sbi.FeatureCompat &^= EROFS_FEATURE_COMPAT_XATTR_FILTER
}
