package types

// lz4_0padding
func (sbi *SuperBlkInfo) ErofsSbHasLz40Padding() bool {
	return sbi.FeatureIncompat&EROFS_FEATURE_INCOMPAT_ZERO_PADDING != 0
}
func (sbi *SuperBlkInfo) ErofsSbSetLz40Padding() {
	sbi.FeatureIncompat |= EROFS_FEATURE_INCOMPAT_ZERO_PADDING
}
func (sbi *SuperBlkInfo) ErofsSbClearLz40Padding() {
	sbi.FeatureIncompat &^= EROFS_FEATURE_INCOMPAT_ZERO_PADDING
}

// compr_cfgs
func (sbi *SuperBlkInfo) ErofsSbHasComprCfgs() bool {
	return sbi.FeatureIncompat&EROFS_FEATURE_INCOMPAT_COMPR_CFGS != 0
}
func (sbi *SuperBlkInfo) ErofsSbSetComprCfgs() {
	sbi.FeatureIncompat |= EROFS_FEATURE_INCOMPAT_COMPR_CFGS
}
func (sbi *SuperBlkInfo) ErofsSbClearComprCfgs() {
	sbi.FeatureIncompat &^= EROFS_FEATURE_INCOMPAT_COMPR_CFGS
}

// big_pcluster
func (sbi *SuperBlkInfo) ErofsSbHasBigPcluster() bool {
	return sbi.FeatureIncompat&EROFS_FEATURE_INCOMPAT_BIG_PCLUSTER != 0
}
func (sbi *SuperBlkInfo) ErofsSbSetBigPcluster() {
	sbi.FeatureIncompat |= EROFS_FEATURE_INCOMPAT_BIG_PCLUSTER
}
func (sbi *SuperBlkInfo) ErofsSbClearBigPcluster() {
	sbi.FeatureIncompat &^= EROFS_FEATURE_INCOMPAT_BIG_PCLUSTER
}

// chunked_file
func (sbi *SuperBlkInfo) ErofsSbHasChunkedFile() bool {
	return sbi.FeatureIncompat&EROFS_FEATURE_INCOMPAT_CHUNKED_FILE != 0
}
func (sbi *SuperBlkInfo) ErofsSbSetChunkedFile() {
	sbi.FeatureIncompat |= EROFS_FEATURE_INCOMPAT_CHUNKED_FILE
}
func (sbi *SuperBlkInfo) ErofsSbClearChunkedFile() {
	sbi.FeatureIncompat &^= EROFS_FEATURE_INCOMPAT_CHUNKED_FILE
}

// device_table
func (sbi *SuperBlkInfo) ErofsSbHasDeviceTable() bool {
	return sbi.FeatureIncompat&EROFS_FEATURE_INCOMPAT_DEVICE_TABLE != 0
}
func (sbi *SuperBlkInfo) ErofsSbSetDeviceTable() {
	sbi.FeatureIncompat |= EROFS_FEATURE_INCOMPAT_DEVICE_TABLE
}
func (sbi *SuperBlkInfo) ErofsSbClearDeviceTable() {
	sbi.FeatureIncompat &^= EROFS_FEATURE_INCOMPAT_DEVICE_TABLE
}

// ztailpacking
func (sbi *SuperBlkInfo) ErofsSbHasZtailpacking() bool {
	return sbi.FeatureIncompat&EROFS_FEATURE_INCOMPAT_ZTAILPACKING != 0
}
func (sbi *SuperBlkInfo) ErofsSbSetZtailpacking() {
	sbi.FeatureIncompat |= EROFS_FEATURE_INCOMPAT_ZTAILPACKING
}
func (sbi *SuperBlkInfo) ErofsSbClearZtailpacking() {
	sbi.FeatureIncompat &^= EROFS_FEATURE_INCOMPAT_ZTAILPACKING
}

// fragments
func (sbi *SuperBlkInfo) ErofsSbHasFragments() bool {
	return sbi.FeatureIncompat&EROFS_FEATURE_INCOMPAT_FRAGMENTS != 0
}
func (sbi *SuperBlkInfo) ErofsSbSetFragments() {
	sbi.FeatureIncompat |= EROFS_FEATURE_INCOMPAT_FRAGMENTS
}
func (sbi *SuperBlkInfo) ErofsSbClearFragments() {
	sbi.FeatureIncompat &^= EROFS_FEATURE_INCOMPAT_FRAGMENTS
}

// dedupe
func (sbi *SuperBlkInfo) ErofsSbHasDedupe() bool {
	return sbi.FeatureIncompat&EROFS_FEATURE_INCOMPAT_DEDUPE != 0
}
func (sbi *SuperBlkInfo) ErofsSbSetDedupe() {
	sbi.FeatureIncompat |= EROFS_FEATURE_INCOMPAT_DEDUPE
}
func (sbi *SuperBlkInfo) ErofsSbClearDedupe() {
	sbi.FeatureIncompat &^= EROFS_FEATURE_INCOMPAT_DEDUPE
}

// xattr_prefixes
func (sbi *SuperBlkInfo) ErofsSbHasXattrPrefixes() bool {
	return sbi.FeatureIncompat&EROFS_FEATURE_INCOMPAT_XATTR_PREFIXES != 0
}
func (sbi *SuperBlkInfo) ErofsSbSetXattrPrefixes() {
	sbi.FeatureIncompat |= EROFS_FEATURE_INCOMPAT_XATTR_PREFIXES
}
func (sbi *SuperBlkInfo) ErofsSbClearXattrPrefixes() {
	sbi.FeatureIncompat &^= EROFS_FEATURE_INCOMPAT_XATTR_PREFIXES
}

// sb_chksum
func (sbi *SuperBlkInfo) ErofsSbHasSbChksum() bool {
	return sbi.FeatureCompat&EROFS_FEATURE_COMPAT_SB_CHKSUM != 0
}
func (sbi *SuperBlkInfo) ErofsSbSetSbChksum() {
	sbi.FeatureCompat |= EROFS_FEATURE_COMPAT_SB_CHKSUM
}
func (sbi *SuperBlkInfo) ErofsSbClearSbChksum() {
	sbi.FeatureCompat &^= EROFS_FEATURE_COMPAT_SB_CHKSUM
}

// xattr_filter
func (sbi *SuperBlkInfo) ErofsSbHasXattrFilter() bool {
	return sbi.FeatureCompat&EROFS_FEATURE_COMPAT_XATTR_FILTER != 0
}
func (sbi *SuperBlkInfo) ErofsSbSetXattrFilter() {
	sbi.FeatureCompat |= EROFS_FEATURE_COMPAT_XATTR_FILTER
}
func (sbi *SuperBlkInfo) ErofsSbClearXattrFilter() {
	sbi.FeatureCompat &^= EROFS_FEATURE_COMPAT_XATTR_FILTER
}
