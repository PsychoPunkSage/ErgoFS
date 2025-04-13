package types

import (
	"fmt"
	"math"
	"syscall"
	"unsafe"

	errs "github.com/PsychoPunkSage/ErgoFS/pkg/errors"
)

var inodeHashtable [NR_INODE_HASHTABLE]ListHead

type ErofsDentry struct {
	DChild ListHead
	// Union using interface{} - can hold either `*ErofsInode` or `uint64`
	// Use type assertion to access the concrete type:
	// if inode, ok := dentry.Entry.(*ErofsInode); ok { ... }
	// if nid, ok := dentry.Entry.(uint64); ok { ... }
	Entry    interface{}
	Namelen  uint8
	Type     uint8
	ValidNid bool
	Name     string
}

func ErofsInodeManagerInit() {
	for i := 0; i < NR_INODE_HASHTABLE; i++ {
		InitListHead(&inodeHashtable[i])
	}
}

func ErofsFlushPackedInode(sbi *SuperBlkInfo) int {
	var epi *ErofsPackedInode
	var inode *ErofsInode

	epi = sbi.PackedInode

	if epi == nil || !ErofsSbHasFragments(sbi) {
		return -errs.EINVAL
	}

	if lseek(epi.Fd, 0, SEEK_SET) <= 0 {
		return 0
	}

	inode, _ = ErofsMkfsBuildSpecialFromFd(sbi, epi.Fd, EROFS_PACKED_INODE)
	sbi.PackedNid = ErofsLookupNid(inode) // priv
	ErofsIput(inode)

	return 0
}

func ErofsMkfsBuildSpecialFromFd(sbi *SuperBlkInfo, fd int, name string) (*ErofsInode, error) {
	var st syscall.Stat_t
	var inode *ErofsInode
	var ictx interface{}

	// Seek to the beginning of the file
	_, err := syscall.Seek(fd, 0, SEEK_SET)
	if err != nil {
		return nil, err
	}

	// Get file stats
	err = syscall.Fstat(fd, &st)
	if err != nil {
		return nil, err
	}

	inode = ErofsNewInode(sbi)
	// Error handling:)

	if name == EROFS_PACKED_INODE {
		st.Uid = 0
		st.Gid = 0
		st.Nlink = 0
	}

	// Fill the inode with file stats and name
	err = ErofsFillInode(inode, &st, name)
	if err != nil {
		return nil, err
	}

	// Additional handling for packed inodes
	if name == EROFS_PACKED_INODE {
		inode.Sbi.PackedNid = EROFS_PACKED_NID_UNALLOCATED
		inode.Nid = inode.Sbi.PackedNid
	}

	// Check if compression is enabled and file is compressible
	if len(GCfg.CompressionOptions) > 0 && GCfg.CompressionOptions[0].Algorithm != "" && ErofsFileIsCompressible(inode) {
		ictx, err = ErofsBeginCompressedFile(inode, fd, 0)
		if err != nil {
			return nil, err
		}

		if ictx == nil {
			panic("context should not be nil")
		}

		err = ErofsWriteCompressedFile(ictx)
		if err == nil {
			goto out
		}

		// If error is not ENOSPC, return error
		if err != syscall.ENOSPC {
			return nil, err
		}

		// Try to seek back to beginning for uncompressed write
		_, err = syscall.Seek(fd, 0, SEEK_SET)
		if err != nil {
			return nil, err
		}
	}

	// Write uncompressed file
	err = WriteUncompressedFileFromFd(inode, fd)
	if err != nil {
		return nil, err
	}

out:
	ErofsPrepareInodeBuffer(inode)
	ErofsWriteTailEnd(inode)
	return inode, nil
}

func ErofsFillInode(inode *ErofsInode, st *syscall.Stat_t, path string) error {
	err := erofsFillInode(inode, st, path)
	if err != nil {
		return err
	}

	inode.IMode = uint16(st.Mode)
	inode.INlink = 1

	switch inode.IMode & syscall.S_IFMT {
	case syscall.S_IFCHR:
	case syscall.S_IFBLK:
	case syscall.S_IFIFO:
	case syscall.S_IFSOCK:
		inode.IRdev = erofsNewEncodeDev(st.Rdev)
	case syscall.S_IFDIR:
		inode.ISize = 0
		break
	case syscall.S_IFREG:
	case syscall.S_IFLNK:
		inode.ISize = uint64(st.Size)
		break
	default:
		return syscall.Errno(errs.EINVAL)
	}

	inode.ISrcpath = path

	if ErofsShouldUseInodeExtended(inode) {
		if GCfg.ForceInodeVersion == FORCE_INODE_COMPACT {
			fmt.Errorf("file %s cannot be in compact form", inode.ISrcpath)
			return syscall.Errno(errs.EINVAL)
		}
		inode.InodeIsize = uint8(unsafe.Sizeof(ErofsInodeExtended{}))
	} else {
		inode.InodeIsize = uint8(unsafe.Sizeof(ErofsInodeCompact{}))
	}

	inode.Dev = uint32(st.Dev)
	inode.IIno[1] = st.Ino

	ErofsInsertIhash(inode)
	return nil
}

func ErofsNewInode(sbi *SuperBlkInfo) *ErofsInode {
	var inode *ErofsInode

	inode.Sbi = sbi
	inode.ICount = 1
	inode.DataLayout = EROFS_INODE_FLAT_PLAIN

	InitListHead(&inode.IHash)
	InitListHead(&inode.ISubdirs)
	InitListHead(&inode.IXattrs)
	return inode
}

func ErofsLookupNid(inode *ErofsInode) uint64 {
	var off, metaOffset uint64
	bh := inode.Bh
	sbi := inode.Sbi

	if bh != nil && inode.Nid <= 0 {
		MapBh(nil, bh.Block)
		off = BhTell(bh, false)

		metaOffset = ErofsPos(sbi, uint64(sbi.MetaBlkAddr))
		if !(off < metaOffset) { // DBG_BUGON equivalent
			panic("Bug: off < metaOffset")
		}

		inode.Nid = (off - metaOffset) >> EROFSISLOTBITS
	}

	if IS_ROOT(inode) && inode.Nid > 0xffff {
		return uint64(sbi.RootNid)
	}

	return inode.Nid
}

func ErofsIput(inode *ErofsInode) uint {
	got := ErofsAtomicDecReturn(&inode.ICount)

	if got >= 1 {
		return got
	}

	// Using your existing ListForEachInListSafe function
	ListForEachInListSafe(func(pos, n *ListHead) bool {
		_ = ErofsDentryFromList(pos)
		// Process dentry here
		// No need to free in Go, just remove references if needed
		return true // Continue iteration
	}, &inode.ISubdirs)

	// In Go we don't explicitly free memory, but we clear references
	// to assist garbage collection
	inode.Compressmeta = nil
	inode.EofTailraw = nil

	// Remove this inode from hash list
	ListDel(&inode.IHash)

	inode.ISrcpath = ""

	// Handle resources based on data source type
	if inode.DataSource == EROFS_INODE_DATA_SOURCE_DISKBUF {
		// Close any open resources
		ErofsDiskbufClose(inode.IDiskbuf)
		inode.IDiskbuf = nil
	} else {
		inode.ILink = ""
	}

	// 0: inode has been fully released
	return 0
}

func erofsFillInode(inode *ErofsInode, st *syscall.Stat_t, path string) error {
	err := erofsDroidInodeFsconfig(inode, st, path)
	sbi := inode.Sbi

	if err != nil {
		return err
	}

	inode.IUid = func() uint32 {
		if GCfg.Uid == -1 {
			return st.Uid
		}
		return uint32(GCfg.Uid)
	}()

	inode.IGid = func() uint32 {
		if GCfg.Gid == -1 {
			return st.Uid
		}
		return uint32(GCfg.Gid)
	}()

	if inode.IUid+uint32(GCfg.UidOffset) < 0 { // How is this even possible
		return fmt.Errorf("EROFS: uid overflow")
	}
	inode.IUid += uint32(GCfg.UidOffset)
	if inode.IGid+uint32(GCfg.GidOffset) < 0 { // How is this even possible
		return fmt.Errorf("EROFS: gid overflow")
	}
	inode.IGid += uint32(GCfg.GidOffset)

	inode.IMtime = uint64(st.Mtim.Sec)
	inode.IMtimeNsec = uint32(st.Mtim.Nsec)

	switch GCfg.TimeInherit {
	case TIMESTAMP_CLAMPING:
		if inode.IMtime < sbi.BuildTime {
			break
		}
	case TIMESTAMP_FIXED:
		inode.IMtime = sbi.BuildTime
		inode.IMtimeNsec = sbi.BuildTimeNsec
	default:
		break
	}

	return nil
}

func erofsNewEncodeDev(dev uint64) uint32 {
	maj := Major(dev)
	min := Minor(dev)
	return (min & 0xff) | (maj << 8) | ((min &^ 0xff) << 12)
}

func ErofsShouldUseInodeExtended(inode *ErofsInode) bool {
	if GCfg.ForceInodeVersion == FORCE_INODE_EXTENDED {
		return true
	}

	if inode.ISize > math.MaxUint32 {
		return true
	}

	if erofsIsPackedInode(inode) {
		return false
	}

	if inode.IUid > math.MaxUint16 {
		return true
	}

	if inode.IGid > math.MaxUint16 {
		return true
	}

	if inode.INlink > math.MaxUint16 {
		return true
	}

	if (inode.IMtime != inode.Sbi.BuildTime || inode.IMtimeNsec != inode.Sbi.BuildTimeNsec) && (!GCfg.IgnoreMtime) {
		return true
	}

	return false
}

func ErofsInsertIhash(inode *ErofsInode) {
	nr := (inode.IIno[1] ^ uint64(inode.Dev)) % NR_INODE_HASHTABLE
	ListAdd(&inode.IHash, &inodeHashtable[nr])
}

func ErofsFileIsCompressible(inode *ErofsInode) bool {
	if GCfg.CompressHintsFile != "" {
		return zErodsApplyCompressHints(inode)
	}
	return true
}

func erofsIsPackedInode(inode *ErofsInode) bool {
	packedNid := inode.Sbi.PackedNid

	if inode.Nid == EROFS_PACKED_NID_UNALLOCATED {
		if packedNid != EROFS_PACKED_NID_UNALLOCATED {
			panic("packedNid should be unallocated")
		}
		return true
	}

	return packedNid > 0 && inode.Nid == packedNid
}

/*
#ifdef WITH_ANDROID
int erofs_droid_inode_fsconfig(struct erofs_inode *inode,

	struct stat *st,
	const char *path)

	{
		mode_t stat_file_type_mask = st->st_mode & S_IFMT;
		unsigned int uid = 0, gid = 0, mode = 0;
		const char *fspath;
		char *decorated = NULL;

		inode->capabilities = 0;
		if (!cfg.fs_config_file && !cfg.mount_point)
			return 0;
		if (path == EROFS_PACKED_INODE)
			return 0;

		if (!cfg.mount_point ||
		    (cfg.fs_config_file && erofs_fspath(path)[0] == '\0')) {
			fspath = erofs_fspath(path);
		} else {
			if (asprintf(&decorated, "%s/%s", cfg.mount_point,
				     erofs_fspath(path)) <= 0)
				return -ENOMEM;
			fspath = decorated;
		}

		if (cfg.fs_config_file)
			canned_fs_config(fspath, S_ISDIR(st->st_mode),
					 cfg.target_out_path,
					 &uid, &gid, &mode, &inode->capabilities);
		else
			fs_config(fspath, S_ISDIR(st->st_mode),
				  cfg.target_out_path,
				  &uid, &gid, &mode, &inode->capabilities);

		erofs_dbg("/%s -> mode = 0x%x, uid = 0x%x, gid = 0x%x, capabilities = 0x%" PRIx64,
			  fspath, mode, uid, gid, inode->capabilities);

		if (decorated)
			free(decorated);
		st->st_uid = uid;
		st->st_gid = gid;
		st->st_mode = mode | stat_file_type_mask;
		return 0;
	}

#else
static int erofs_droid_inode_fsconfig(struct erofs_inode *inode,

	struct stat *st,
	const char *path)

	{
		return 0;
	}

#endif
*/
func erofsDroidInodeFsconfig(inode *ErofsInode, st syscall.Stat_t, path string) error {
	return nil
}
