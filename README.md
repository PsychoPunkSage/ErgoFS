# ErgoFS

## EroFS Filesystem

<details>
<summary>Filesystem</summary>

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

</details>

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
[INFO] Creating EROFS filesystem
[INFO] Source dir: test_data
[INFO] Output file: test.img
[INFO] Block size: 4096
[DEBUG] Block size bits: 12
[INFO] Opening output file
[INFO] Creating root directory inode
[INFO] Building filesystem from: test_data
[INFO] Writing inodes to disk
[INFO] Finalizing and closing filesystem
[INFO] Finalizing EROFS filesystem:
[INFO] Magic:            0xe0f5e1e0
[INFO] Block Size:       4096 bytes (ilog2: 12)
[INFO] Feature Compat:   0x0001
[INFO] Feature Incompat: 0x0001
[INFO] Inode Count:      6
[INFO] Blocks:           258
[INFO] Current Position: 1053120
[INFO] Checksum:         0x7dcb8f03
[DEBUG] Superblock binary representation:
SB: e0 e1 f5 e0 00 0c 01 00 00 00 01 00 00 00 01 02 
SB: 03 04 05 06 07 08 09 0a 0b 0c 0d 0e 0f 10 00 00 
SB: 00 00 00 00 00 00 00 00 00 00 00 00 00 00 02 01 
SB: 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 
[INFO] Wrote 146 bytes for superblock
[INFO] Verification of written magic: 0xe0f5e1e0
[DEBUG] First 256 bytes of the image file:
IMG: e0 e1 f5 e0 00 0c 01 00 00 00 01 00 00 00 01 02 
IMG: 03 04 05 06 07 08 09 0a 0b 0c 0d 0e 0f 10 00 00 
IMG: 00 00 00 00 00 00 00 00 00 00 00 00 00 00 02 01 
IMG: 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 
IMG: 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 
IMG: 00 00 06 00 00 00 00 00 00 00 72 4d c8 67 00 00 
IMG: 00 00 a1 a9 b9 1b 00 00 00 00 03 8f cb 7d 00 00 
IMG: 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 
IMG: 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 
IMG: 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 
IMG: 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 
IMG: 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 
IMG: 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 
IMG: 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 
IMG: 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 
IMG: 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 
Successfully created EROFS image: test.img
Verifying image before fix...
Raw Superblock Data (first 32 bytes):
e0 e1 f5 e0 00 0c 01 00 00 00 01 00 00 00 01 02 
03 04 05 06 07 08 09 0a 0b 0c 0d 0e 0f 10 00 00 

Magic number in hex: 0xe0f5e1e0
Expected magic:      0xe0f5e1e0

EROFS Image Information:
Magic:            0xe0f5e1e0
Checksum Alg:     0
Block Size:       1 bytes (ilog2: 0)
SB Extension Ver: 0
Feature Compat:   0x0001
Feature Incompat: 0x0001
UUID:             00000102030405060708090a0b0c0d0e
Volume Name:      
Inode Count:      16908288
Blocks:           0
Meta Block Addr:  0x00000000
Xattr Block Addr: 0x00000000
Extra Devices:    0
Build Time:       0.000000000
Checksum:         0x00000000

File Size:        1053120 bytes
Expected Blocks:  1053120
WARNING: Block count mismatch. Superblock says 0 blocks, file size suggests 1053120 blocks.

Attempting to read root inode metadata...
First 32 bytes of root inode area:
e1 f5 e0 00 0c 01 00 00 00 01 00 00 00 01 02 03 
04 05 06 07 08 09 0a 0b 0c 0d 0e 0f 10 00 00 00 

Verification complete!
Verifying image after fix...
Raw Superblock Data (first 32 bytes):
e0 e1 f5 e0 00 0c 01 00 00 00 01 00 00 00 01 02 
03 04 05 06 07 08 09 0a 0b 0c 0d 0e 0f 10 00 00 

Magic number in hex: 0xe0f5e1e0
Expected magic:      0xe0f5e1e0

EROFS Image Information:
Magic:            0xe0f5e1e0
Checksum Alg:     0
Block Size:       1 bytes (ilog2: 0)
SB Extension Ver: 0
Feature Compat:   0x0001
Feature Incompat: 0x0001
UUID:             00000102030405060708090a0b0c0d0e
Volume Name:      
Inode Count:      16908288
Blocks:           0
Meta Block Addr:  0x00000000
Xattr Block Addr: 0x00000000
Extra Devices:    0
Build Time:       0.000000000
Checksum:         0x00000000

File Size:        1053120 bytes
Expected Blocks:  1053120
WARNING: Block count mismatch. Superblock says 0 blocks, file size suggests 1053120 blocks.

Attempting to read root inode metadata...
First 32 bytes of root inode area:
e1 f5 e0 00 0c 01 00 00 00 01 00 00 00 01 02 03 
04 05 06 07 08 09 0a 0b 0c 0d 0e 0f 10 00 00 00 

Verification complete!
Creating mount point...
Attempting to mount the image...
[sudo] password for psychopunk_sage: 
mount: /home/psychopunk_sage/dev/OpenSource/Unikraft/ErgoFS/mount_point: wrong fs type, bad option, bad superblock on /dev/loop48, missing codepage or helper program, or other error.
       dmesg(1) may have more information after failed mount system call.
Mount failed. Checking kernel messages...
[ 2828.829917] br0: port 33(veth2002) entered forwarding state
[ 2828.842057] eth2: renamed from veth4734d59
[ 2828.843113] docker_gwbridge: port 49(veth076cfe2) entered blocking state
[ 2828.843123] docker_gwbridge: port 49(veth076cfe2) entered forwarding state
[ 2828.896627] veth2005: renamed from veth07d5783
[ 2828.897664] br0: port 52(veth2005) entered blocking state
[ 2828.897675] br0: port 52(veth2005) entered disabled state
[ 2828.897740] veth2005: entered allmulticast mode
[ 2828.897896] veth2005: entered promiscuous mode
[ 2828.925100] docker_gwbridge: port 51(veth7f3e78a) entered blocking state
[ 2828.925112] docker_gwbridge: port 51(veth7f3e78a) entered disabled state
[ 2828.925189] veth7f3e78a: entered allmulticast mode
[ 2828.930556] veth7f3e78a: entered promiscuous mode
[ 2828.944593] loop48: detected capacity change from 0 to 2056
[ 2829.015916] erofs: (device loop48): erofs_read_superblock: cannot find valid erofs superblock
[ 2829.042691] veth2003: renamed from vethe1407bc
[ 2829.043747] br0: port 43(veth2003) entered blocking state
[ 2829.043760] br0: port 43(veth2003) entered disabled state
[ 2829.044482] veth2003: entered allmulticast mode
[ 2829.045289] veth2003: entered promiscuous mode
===== Test Complete =====
```

</details>