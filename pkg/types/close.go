package types

import "fmt"

// ///////////
// / CLOSE ///
// ///////////

// static struct erofs_diskbufstrm {
// 	erofs_atomic_t count;
// 	u64 tailoffset, devpos;
// 	int fd;
// 	unsigned int alignsize;
// 	bool locked;
// } *dbufstrm;

type ErofsDiskBufStrm struct {
	count      uint64
	TailOffset uint64
	DevPos     uint64
	Fd         int
	AlignSize  uint
	locked     bool
}

func ErofsDiskbufClose(diskBuf *ErofsDiskbuf) {
	var strm *ErofsDiskBufStrm

	strm = (*ErofsDiskBufStrm)(diskBuf.Sp)

	if strm == nil {
		fmt.Println("strm is Empty")
	}
	// 	DBG_BUGON(erofs_atomic_read(&strm->count) <= 1);
	ErofsAtomicDecReturn(&strm.count) // 	(void)erofs_atomic_dec_return(&strm->count);
	diskBuf.Sp = nil
}
