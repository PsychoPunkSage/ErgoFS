# ErgoFS - Go impl of EroFS

## EroFS Filesystem

<details>
<summary>Filesystem (C lang impl )</summary>

```
.
├── autogen.sh
├── include
│   ├── erofs
│   │   ├── atomic.h
│   │   ├── bitops.h
│   │   ├── blobchunk.h
│   │   ├── block_list.h
│   │   ├── cache.h
│   │   ├── compress.h
│   │   ├── compress_hints.h
│   │   ├── config.h
│   │   ├── decompress.h
│   │   ├── dedupe.h
│   │   ├── defs.h
│   │   ├── dir.h
│   │   ├── diskbuf.h
│   │   ├── err.h
│   │   ├── exclude.h
│   │   ├── flex-array.h
│   │   ├── fragments.h
│   │   ├── hashmap.h
│   │   ├── hashtable.h
│   │   ├── inode.h
│   │   ├── internal.h
│   │   ├── io.h
│   │   ├── list.h
│   │   ├── print.h
│   │   ├── rebuild.h
│   │   ├── tar.h
│   │   ├── trace.h
│   │   ├── workqueue.h
│   │   └── xattr.h
│   └── erofs_fs.h
├── lib
│   ├── bitops.c
│   ├── blobchunk.c
│   ├── block_list.c
│   ├── cache.c
│   ├── compress.c
│   ├── compress_hints.c
│   ├── compressor.c
│   ├── compressor_deflate.c
│   ├── compressor.h
│   ├── compressor_libdeflate.c
│   ├── compressor_liblzma.c
│   ├── compressor_libzstd.c
│   ├── compressor_lz4.c
│   ├── compressor_lz4hc.c
│   ├── config.c
│   ├── data.c
│   ├── decompress.c
│   ├── dedupe.c
│   ├── dir.c
│   ├── diskbuf.c
│   ├── exclude.c
│   ├── fragments.c
│   ├── hashmap.c
│   ├── inode.c
│   ├── io.c
│   ├── kite_deflate.c
│   ├── liberofs_private.h
│   ├── liberofs_uuid.h
│   ├── liberofs_xxhash.h
│   ├── Makefile.am
│   ├── namei.c
│   ├── rebuild.c
│   ├── rolling_hash.h
│   ├── sha256.c
│   ├── sha256.h
│   ├── super.c
│   ├── tar.c
│   ├── uuid.c
│   ├── uuid_unparse.c
│   ├── workqueue.c
│   ├── xattr.c
│   ├── xxhash.c
│   └── zmap.c
├── Makefile.am
├── mkfs
│   ├── main.c
│   └── Makefile.am
├── README
└── VERSION

13 directories, 104 files
```

</details><br>

<details>
<summary>My file str</summary>

### Disclaimer
> this str is temporary and will be changed as this project gets more mature

```
.
├── cmd
│   ├── mkfs
│   │   └── main.go
│   └── verify
│       └── main.go
├── c_reference.img
├── go.mod
├── go_reference.img
├── mkfs.erofs
├── mount_point
├── pkg
│   ├── compression
│   │   ├── algorithms.go
│   │   ├── compress.go
│   │   └── lz4.go
│   ├── dedupe
│   │   └── dedupe.go
│   ├── types
│   │   ├── config.go
│   │   ├── constants.go
│   │   ├── debug.go
│   │   ├── inode.go
│   │   └── superblock.go
│   ├── util
│   │   ├── blocklist.go
│   │   ├── buffer.go
│   │   └── diskbuf.go
│   └── writer
│       ├── builder.go
│       ├── inode.go
│       ├── utils.go
│       └── xattr.go
├── README.md
├── report
│   ├── c_hexdump.txt
│   ├── c_superblock.bin
│   ├── c_superblock.txt
│   ├── diff.log
│   ├── diff_superblock.log
│   ├── go_hexdump.txt
│   ├── go_superblock.bin
│   └── go_superblock.txt
├── report.sh
├── script.sh
├── test
└── test_data
    ├── hello.txt
    ├── random.bin
    ├── subdir
    │   └── test.txt
    └── symlink -> hello.txt

15 directories, 37 files
```

</details>

### Project Structure (for now)

* `pkg/types/`: Core data structures and constants
* `pkg/writer/`: Filesystem creation functionality
* `cmd/mkfs/`: Command line interface

## How to use?

```sh
./script.sh
```

<details>
<summary>OUTPUT(Go-ErgoFS)</summary>

```
===== EROFS Testing Script =====
Creating test data...
Building tools...
Creating EROFS image...
Creating EroFS filesystem on test.img
Block size: 4096 bytes
Input directory: /home/psychopunk_sage/dev/OpenSource/Unikraft/erofs-utils/test_data
Found 3 files/directories
Superblock checksum: 0xec1f893c
EroFS filesystem creation complete
Total blocks: 0
Total inodes: 3
Creating mount point...
Attempting to mount the image...
mount: /home/psychopunk_sage/dev/OpenSource/Unikraft/ErgoFS/mount_point: wrong fs type, bad option, bad superblock on /dev/loop48, missing codepage or helper program, or other error.
       dmesg(1) may have more information after failed mount system call.
Mount failed. Checking kernel messages...
[56389.072955] wlo1: disconnect from AP e8:10:98:6b:3e:31 for new auth to e8:10:98:6b:2e:51
[56389.244603] wlo1: authenticate with e8:10:98:6b:2e:51 (local address=f8:89:d2:8d:e7:05)
[56389.244611] wlo1: send auth to e8:10:98:6b:2e:51 (try 1/3)
[56389.356822] wlo1: send auth to e8:10:98:6b:2e:51 (try 2/3)
[56389.357976] wlo1: authenticated
[56389.358510] wlo1: associate with e8:10:98:6b:2e:51 (try 1/3)
[56389.362274] wlo1: RX ReassocResp from e8:10:98:6b:2e:51 (capab=0x1411 status=0 aid=1)
[56389.477200] wlo1: associated
[56389.477309] wlo1: Limiting TX power to 36 (36 - 0) dBm as advertised by e8:10:98:6b:2e:51
[56500.425821] wlo1: disconnect from AP e8:10:98:6b:2e:51 for new auth to e8:10:98:6b:2e:41
[56500.595553] wlo1: authenticate with e8:10:98:6b:2e:41 (local address=f8:89:d2:8d:e7:05)
[56500.595561] wlo1: send auth to e8:10:98:6b:2e:41 (try 1/3)
[56500.708576] wlo1: send auth to e8:10:98:6b:2e:41 (try 2/3)
[56500.710748] wlo1: authenticated
[56500.711536] wlo1: associate with e8:10:98:6b:2e:41 (try 1/3)
[56500.715043] wlo1: RX ReassocResp from e8:10:98:6b:2e:41 (capab=0x1531 status=0 aid=1)
[56500.828833] wlo1: associated
[56500.828901] wlo1: Limiting TX power to 36 (36 - 0) dBm as advertised by e8:10:98:6b:2e:41
[56616.930325] loop48: detected capacity change from 0 to 8
[56616.930607] erofs: (device loop48): erofs_read_superblock: dirblkbits 12 isn't supported
===== Test Complete =====
```

</details>

<hr>

```sh
./script.sh
```

<details>
<summary>OUTPUT(C-EroFS)</summary>

```
mkfs.erofs 1.8.5-g6fa861e2-dirty
<I> erofs_io: successfully to open c_reference.img
	c_version:           [1.8.5-g6fa861e2-dirty]
	c_dbg_lvl:           [       4]
	c_dry_run:           [       0]
<I> erofs: file / dumped (mode 40775)
<I> erofs: file /hello.txt dumped (mode 100664)
<I> erofs: file /random.bin dumped (mode 100664)
<I> erofs: file /subdir dumped (mode 40775)
<I> erofs: file /symlink dumped (mode 120777)
<I> erofs: file /subdir/test.txt dumped (mode 100664)
<I> erofs: superblock checksum 0x1be3f68f written
------
Filesystem UUID: 7dd00173-4052-4115-9b79-fcca49d178c1
Filesystem total blocks: 257 (of 4096-byte blocks)
Filesystem total inodes: 6
Filesystem total metadata blocks: 1
Filesystem total deduplicated bytes (of source files): 0
```

</details>

## Hardblocks

getting this `[59132.367167] erofs: (device loop48): erofs_read_superblock: dirblkbits 12 isn't supported` 
* this is weird cause I'm not hardonding anything to 12 yet I'm getting same error over and over again

You can have a look [@Report](./report/) (I have compared images I got from C and Go based EroFS)

Not sure where the issue lies