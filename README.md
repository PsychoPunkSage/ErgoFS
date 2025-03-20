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
│   └── mkfs
│       └── main.go
├── compress_hints.txt
├── go.mod
├── go.sum
├── pkg
│   ├── compression
│   │   ├── algorithms.go
│   │   ├── compress.go
│   │   └── lz4.go
│   ├── dedupe
│   │   └── dedupe.go
│   ├── types
│   │   ├── buffer.go
│   │   ├── compress.go
│   │   ├── compressor
│   │   ├── config.go
│   │   ├── constants.go
│   │   ├── debug.go
│   │   ├── error.go
│   │   ├── helper.go
│   │   ├── inode.go
│   │   ├── internal.go
│   │   ├── list.go
│   │   ├── lz4.go
│   │   ├── macros.go
│   │   ├── superblock.go
│   │   └── uuid.go
│   ├── util
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
├── run.sh
├── script.sh
├── test
└── test_data
    ├── hello.txt
    ├── random.bin
    ├── subdir
    │   └── test.txt
    └── symlink -> hello.txt

15 directories, 48 files
```

</details>

### Project Structure (for now)

* `pkg/types/`: Core data structures and constants
* `pkg/writer/`: Filesystem creation functionality
* `cmd/mkfs/`: Command line interface

## How to use?

```sh
./run.sh
```

<details>
<summary>OUTPUT(Go-ErgoFS)</summary>

```
Debug Level: 4, Image Path: output.img, Source Path: test_data/
I'm here
successfully opened output.img
Superblock reserved successfully: &{{0xc000104200 0xc000104200} 0xc000112070 0 0x5e01e0 <nil>}
UUID generated successfully: [51 56 74 108 149 254 78 116 130 208 58 195 69 200 222 149]
Compress Initialization successfully Done
Erofs Bflush successfully executed
Superblock successfully Written
ret: 0
SuperBlock checksum 0x5c390b03 written
```

> $ sudo dmesg | grep -i erofs

```
NO OUTPUT :)
```

> $ sudo dmesg

```
[40382.691728] wlo1: associated
[40382.691852] wlo1: Limiting TX power to 30 (30 - 0) dBm as advertised by e8:10:98:6b:3e:32
[40448.887111] audit: type=1400 audit(1742497728.950:517): apparmor="DENIED" operation="open" class="file" profile="snap.brave.brave" name="/proc/pressure/cpu" pid=30524 comm="ThreadPoolForeg" requested_mask="r" denied_mask="r" fsuid=1000 ouid=0
[40448.887131] audit: type=1400 audit(1742497728.950:518): apparmor="DENIED" operation="open" class="file" profile="snap.brave.brave" name="/proc/pressure/io" pid=30524 comm="ThreadPoolForeg" requested_mask="r" denied_mask="r" fsuid=1000 ouid=0
[40448.887140] audit: type=1400 audit(1742497728.950:519): apparmor="DENIED" operation="open" class="file" profile="snap.brave.brave" name="/proc/pressure/memory" pid=30524 comm="ThreadPoolForeg" requested_mask="r" denied_mask="r" fsuid=1000 ouid=0
[40619.188322] wlo1: authenticate with e8:10:98:6b:2e:52 (local address=f8:89:d2:8d:e7:05)
[40619.188338] wlo1: send auth to e8:10:98:6b:2e:52 (try 1/3)
[40619.189880] wlo1: authenticated
[40619.191955] wlo1: associate with e8:10:98:6b:2e:52 (try 1/3)
[40619.196179] wlo1: RX AssocResp from e8:10:98:6b:2e:52 (capab=0x1411 status=0 aid=3)
[40619.305425] wlo1: associated
[40619.305672] wlo1: Limiting TX power to 36 (36 - 0) dBm as advertised by e8:10:98:6b:2e:52
```
* No recent logs are related to superblock errors. (i.e. earlier I got the malformed superblock error... now everything is working fine)

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

</details><br>

<!-- HI
| Function Call                                 | Purpose                           | Reason                                                                  |
| --------------------------------------------- | --------------------------------- | ----------------------------------------------------------------------- |
| - [X] `erofs_init_configure()`                | Initializes the configuration     | Sets up the global configuration environment before parsing any options |
| - [X] `erofs_mkfs_default_options()`          | Sets default options              | Establishes baseline configuration values before command-line overrides |
| - [ ] `mkfs_parse_options_cfg()`              | Parses command-line options       | Processes user-provided parameters to customize the filesystem creation |
| - [X] `erofs_show_progs()`                    | Displays program information      | Shows version info and command-line arguments for diagnostic purposes   |
| - [ ] `parse_source_date_epoch()`             | Parses timestamp information      | Gets source date epoch for reproducible builds if specified             |
| - [ ] `gettimeofday()`                        | Gets current time                 | Sets build timestamp if not explicitly provided                         |
| - [X] `erofs_dev_open()`                      | Opens the output device           | Creates or opens the target filesystem image file                       |
| - [ ] `load_canned_fs_config()`               | Loads filesystem configuration    | Android-specific: loads predefined filesystem configs if specified      |
| - [ ] `erofs_blocklist_open()`                | Opens block list file             | For tracking block allocations if specified                             |
| - [ ] `erofs_show_config()`                   | Shows the configuration           | Displays current configuration for user verification                    |
| - [ ] `erofs_packedfile_init()`               | Initializes packed files support  | Sets up handling for fragments and extended attributes                  |
| - [ ] `erofs_blocklist_open()`                | Opens mapping file in tar mode    | Opens a file to record block mappings when in tar mode                  |
| - [ ] `erofs_read_superblock()`               | Reads existing superblock         | In rebuild/incremental mode: reads superblock from source filesystem    |
| - [ ] `erofs_buffer_init()`                   | Initializes buffer manager        | Sets up the buffer management system for filesystem creation            |
| - [ ] `erofs_reserve_sb()`                    | Reserves space for superblock     | Allocates space for the superblock in the filesystem                    |
| - [ ] `erofs_uuid_generate()`                 | Generates UUID                    | Creates a unique identifier for the filesystem                          |
| - [ ] `erofs_diskbuf_init()`                  | Initializes disk buffer           | Sets up buffers for tar mode operations                                 |
| - [ ] `erofs_load_compress_hints()`           | Loads compression hints           | Reads compression configuration from hint file if specified             |
| - [ ] `z_erofs_compress_init()`               | Initializes compression           | Sets up the compression subsystems based on configuration               |
| - [ ] `z_erofs_dedupe_init()`                 | Initializes deduplication         | Sets up data deduplication if enabled                                   |
| - [ ] `erofs_blob_init()`                     | Initializes blob handling         | Sets up blob device support if configured                               |
| - [ ] `erofs_mkfs_init_devices()`             | Initializes multiple devices      | Sets up device table for multi-device support                           |
| - [ ] `erofs_inode_manager_init()`            | Initializes inode manager         | Prepares the inode management subsystem                                 |
| - [ ] `erofs_rebuild_make_root()`             | Creates root inode                | Makes the root directory inode for tar or rebuild mode                  |
| - [ ] `tarerofs_parse_tar()`                  | Parses tar archive                | Processes input tar archive in tar mode                                 |
| - [ ] `erofs_rebuild_dump_tree()`             | Dumps directory tree              | Writes the rebuilt directory structure to the filesystem                |
| - [ ] `erofs_mkfs_rebuild_load_trees()`       | Loads directory trees             | Loads directory structure from source in rebuild mode                   |
| - [ ] `erofs_build_shared_xattrs_from_path()` | Builds shared extended attributes | Processes extended attributes from the source path                      |
| - [ ] `erofs_xattr_flush_name_prefixes()`     | Flushes xattr name prefixes       | Writes extended attribute name prefixes to the filesystem               |
| - [ ] `erofs_mkfs_build_tree_from_path()`     | Builds directory tree             | Creates filesystem structure from the source directory                  |
| - [ ] `erofs_flush_packed_inode()`            | Flushes packed inodes             | Writes packed data to the filesystem if fragments enabled               |
| - [ ] `erofs_mkfs_dump_blobs()`               | Dumps blobs                       | Writes blob data to the filesystem                                      |
| - [ ] `erofs_bflush()`                        | Flushes buffers                   | Writes all pending buffers to the filesystem except superblock          |
| - [ ] `erofs_fixup_root_inode()`              | Fixes up root inode               | Updates root inode information before writing superblock                |
| - [ ] `erofs_iput()`                          | Releases inode reference          | Cleans up the root inode reference                                      |
| - [ ] `erofs_writesb()`                       | Writes superblock                 | Writes the prepared superblock to the filesystem                        |
| - [ ] `erofs_bflush()`                        | Flushes remaining buffers         | Writes any remaining buffers after superblock                           |
| - [ ] `erofs_dev_resize()`                    | Resizes the device                | Adjusts the final size of the filesystem image                          |
| - [ ] `erofs_enable_sb_chksum()`              | Enables superblock checksum       | Calculates and writes superblock checksum if enabled                    |
| - [ ] `erofs_put_super()`                     | Cleans up superblock              | Frees resources associated with the superblock                          |
| - [ ] Various cleanup functions               | Clean up resources                | Release all allocated resources and clean up subsystems                 |
HiV -->


<!-- ## Hardblocks

getting this `[59132.367167] erofs: (device loop48): erofs_read_superblock: dirblkbits 12 isn't supported` 
* this is weird cause I'm not hardonding anything to 12 yet I'm getting same error over and over again

You can have a look [@Report](./report/) (I have compared images I got from C and Go based EroFS)

Not sure where the issue lies -->