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
├── go.mod
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
│   │   └── diskbuf.go
│   └── writer
│       ├── builder.go
│       ├── inode.go
│       ├── utils.go
│       └── xattr.go
├── README.md
├── script.sh
└── test.img

14 directories, 27 files
```

</details><br>

### Project Structure (for now)

* `pkg/types/`: Core data structures and constants
* `pkg/writer/`: Filesystem creation functionality
* `cmd/mkfs/`: Command line interface

## How to use?

```sh
./script.sh
```

<details>
<summary>OUTPUT</summary>

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

## Hardblocks

getting this `[ 2829.015916] erofs: (device loop48): erofs_read_superblock: cannot find valid erofs superblock`

Not sure where the issue lies