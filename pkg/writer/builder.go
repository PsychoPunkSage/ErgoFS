package writer

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/PsychoPunkSage/ErgoFS/pkg/types"
)

// Builder represents an EROFS filesystem builder
type Builder struct {
	Config     *types.Config
	SuperBlock *types.SuperBlkInfo
	Output     *os.File
	CurrentPos int64
	InodeMap   map[uint64]*types.Inode
	PathMap    map[string]*types.Inode // Add a path-to-inode map
	NextIno    uint64
	Root       *types.Inode
}

// NewBuilder creates a new EROFS filesystem builder
func NewBuilder(config *types.Config) *Builder {
	return &Builder{
		Config:     config,
		SuperBlock: types.DefaultSuperBlkInfo(),
		InodeMap:   make(map[uint64]*types.Inode),
		PathMap:    make(map[string]*types.Inode), // Initialize the path map
		NextIno:    1,                             // Start at inode 1
	}
}

// Open opens the output file
func (b *Builder) Open() error {
	var err error
	b.Output, err = os.Create(b.Config.ImagePath)
	if err != nil {
		return err
	}

	// Reserve space for the superblock
	_, err = b.Output.Write(make([]byte, 128))
	if err != nil {
		return err
	}
	b.CurrentPos = 128

	// Align to block size
	blockSize := uint32(1 << b.SuperBlock.BlksizeBits)
	padding := (blockSize - (uint32(b.CurrentPos) % blockSize)) % blockSize
	if padding > 0 {
		_, err = b.Output.Write(make([]byte, padding))
		if err != nil {
			return err
		}
		b.CurrentPos += int64(padding)
	}

	return nil
}

// CreateRoot creates the root directory inode
func (b *Builder) CreateRoot() error {
	// Create root inode with inode number 1
	stat := types.StatInfo{
		Ino:     1,
		Mode:    0755 | 0040000, // drwxr-xr-x
		Type:    types.EROFS_FT_DIR,
		Uid:     0, // root
		Gid:     0, // root
		Size:    4096,
		ModTime: time.Now(),
		NLink:   2, // . and ..
	}

	b.Root = types.NewInode(1, "/", stat)
	b.InodeMap[1] = b.Root
	b.PathMap["/"] = b.Root // Add root to path map
	b.NextIno = 2

	return nil
}

// NormalizePath normalizes a file path for consistent lookup
func (b *Builder) NormalizePath(path string) string {
	// Ensure paths start with /
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}

	// Clean the path to normalize it
	path = filepath.Clean(path)

	return path
}

// GetOrCreateDirectories ensures all parent directories exist in the filesystem
func (b *Builder) GetOrCreateDirectories(path string) (*types.Inode, error) {
	// If it's the root, return immediately
	if path == "/" {
		return b.Root, nil
	}

	// Check if this path already exists
	normPath := b.NormalizePath(path)
	if inode, exists := b.PathMap[normPath]; exists {
		return inode, nil
	}

	// Get parent directory path
	parentPath := filepath.Dir(normPath)
	if parentPath == "." {
		parentPath = "/"
	}

	// Recursively ensure parent exists
	parent, err := b.GetOrCreateDirectories(parentPath)
	if err != nil {
		return nil, err
	}

	// Create this directory
	dirName := filepath.Base(normPath)
	ino := b.NextIno
	b.NextIno++

	stat := types.StatInfo{
		Ino:     ino,
		Mode:    0755 | 0040000, // drwxr-xr-x
		Type:    types.EROFS_FT_DIR,
		Uid:     0, // root
		Gid:     0, // root
		Size:    4096,
		ModTime: time.Now(),
		NLink:   2, // . and ..
	}

	inode := types.NewInode(ino, dirName, stat)

	// Add to parent and maps
	parent.AddChild(inode)
	b.InodeMap[ino] = inode
	b.PathMap[normPath] = inode

	return inode, nil
}

// BuildFromPath builds the filesystem from a source directory
func (b *Builder) BuildFromPath(sourcePath string) error {
	if b.Root == nil {
		if err := b.CreateRoot(); err != nil {
			return err
		}
	}

	// Walk the directory tree
	return filepath.Walk(sourcePath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip the root directory itself
		if path == sourcePath {
			return nil
		}

		// Get relative path from source root
		relPath, err := filepath.Rel(sourcePath, path)
		if err != nil {
			return err
		}

		// Create inode for this file/directory
		return b.AddFromPath(relPath, path, info)
	})
}

// AddFromPath adds a file or directory to the filesystem
func (b *Builder) AddFromPath(relPath, fullPath string, info os.FileInfo) error {
	// Normalize paths
	normRelPath := b.NormalizePath(relPath)

	// Get or create parent directory
	parentPath := filepath.Dir(normRelPath)
	if parentPath == "." {
		parentPath = "/"
	}

	parent, err := b.GetOrCreateDirectories(parentPath)
	if err != nil {
		return err
	}

	// Skip if it's a directory (we already created it in GetOrCreateDirectories)
	if info.IsDir() {
		return nil
	}

	// Create the inode for files
	ino := b.NextIno
	b.NextIno++

	var inodeType uint8
	mode := uint16(info.Mode() & 0777) // Permission bits

	switch {
	case info.Mode()&os.ModeSymlink != 0:
		inodeType = types.EROFS_FT_SYMLINK
		mode |= 0120000 // Symlink flag
	case info.Mode()&os.ModeDevice != 0:
		if info.Mode()&os.ModeCharDevice != 0 {
			inodeType = types.EROFS_FT_CHRDEV
			mode |= 0020000 // Character device flag
		} else {
			inodeType = types.EROFS_FT_BLKDEV
			mode |= 0060000 // Block device flag
		}
	case info.Mode()&os.ModeNamedPipe != 0:
		inodeType = types.EROFS_FT_FIFO
		mode |= 0010000 // FIFO flag
	case info.Mode()&os.ModeSocket != 0:
		inodeType = types.EROFS_FT_SOCK
		mode |= 0140000 // Socket flag
	default:
		inodeType = types.EROFS_FT_REG_FILE
		mode |= 0100000 // Regular file flag
	}

	// Create stat information
	stat := types.StatInfo{
		Ino:     ino,
		Mode:    mode,
		Type:    inodeType,
		Uid:     0, // Default to root
		Gid:     0, // Default to root
		Size:    uint64(info.Size()),
		ModTime: info.ModTime(),
		NLink:   1,
	}

	// Create the inode
	fileName := filepath.Base(relPath)
	inode := types.NewInode(ino, fileName, stat)
	inode.SourcePath = fullPath

	// Add to parent
	parent.AddChild(inode)

	// Store in inode map and path map
	b.InodeMap[ino] = inode
	b.PathMap[normRelPath] = inode

	return nil
}

// WriteInode writes an inode to the output file
func (b *Builder) WriteInode(inode *types.Inode) (uint32, error) {
	// Calculate block address for this inode
	blockSize := uint32(1 << b.SuperBlock.BlksizeBits)
	blockAddr := uint32(b.CurrentPos) / blockSize

	var inodeData []byte

	// Encode the inode based on size and type
	if inode.Stat.Size > 0xFFFFFFFF || inode.IsDirectory() {
		// Use extended inode format
		inodeData = types.EncodeExtendedInode(inode)
	} else {
		// Use compact inode format
		inodeData = types.EncodeCompactInode(inode)
	}

	// Write the inode
	_, err := b.Output.Write(inodeData)
	if err != nil {
		return 0, err
	}

	// Update position
	b.CurrentPos += int64(len(inodeData))

	// If this is a directory, write directory entries
	if inode.IsDirectory() {
		entries := make([]types.DirEntry, 0, len(inode.Children)+2)

		// Add "." entry (self)
		entries = append(entries, types.DirEntry{
			Nid:      inode.Nid,
			NameLen:  1,
			FileType: types.EROFS_FT_DIR,
			Reserved: 0,
			Name:     ".",
		})

		// Add ".." entry (parent)
		parentNid := uint64(1) // Default to root
		if inode.Parent != nil {
			parentNid = inode.Parent.Nid
		}
		entries = append(entries, types.DirEntry{
			Nid:      parentNid,
			NameLen:  2,
			FileType: types.EROFS_FT_DIR,
			Reserved: 0,
			Name:     "..",
		})

		// Add child entries
		for _, child := range inode.Children {
			entries = append(entries, types.DirEntry{
				Nid:      child.Nid,
				NameLen:  uint16(len(child.Name)),
				FileType: child.Stat.Type,
				Reserved: 0,
				Name:     child.Name,
			})
		}

		// Encode and write directory entries
		dirData := types.EncodeDirEntries(entries)
		_, err = b.Output.Write(dirData)
		if err != nil {
			return 0, err
		}

		// Update position
		b.CurrentPos += int64(len(dirData))
	} else if inode.IsRegular() {
		// For regular files, read the file content and write it
		// For simplicity, we're using a basic approach here
		file, err := os.Open(inode.SourcePath)
		if err != nil {
			return 0, err
		}
		defer file.Close()

		// Copy the file content
		written, err := io.Copy(b.Output, file)
		if err != nil {
			return 0, err
		}

		// Update position
		b.CurrentPos += written
	} else if inode.IsSymlink() {
		// For symlinks, read the target and write it
		target, err := os.Readlink(inode.SourcePath)
		if err != nil {
			return 0, err
		}

		// Write the symlink target
		_, err = b.Output.Write([]byte(target))
		if err != nil {
			return 0, err
		}

		// Update position
		b.CurrentPos += int64(len(target))
	}

	// Align to 4 bytes
	padding := (4 - (b.CurrentPos % 4)) % 4
	if padding > 0 {
		_, err = b.Output.Write(make([]byte, padding))
		if err != nil {
			return 0, err
		}
		b.CurrentPos += padding
	}

	return blockAddr, nil
}

// WriteAllInodes writes all inodes to the filesystem
func (b *Builder) WriteAllInodes() error {
	// Process the root inode first
	rootAddr, err := b.WriteInode(b.Root)
	if err != nil {
		return err
	}
	b.Root.BlockAddr = rootAddr

	// Write all other inodes (except root which we already processed)
	for _, inode := range b.InodeMap {
		if inode.Nid == 1 {
			continue // Skip root
		}

		addr, err := b.WriteInode(inode)
		if err != nil {
			return err
		}
		inode.BlockAddr = addr
	}

	return nil
}

// Close finalizes and closes the filesystem
// func (b *Builder) Close() error {
// 	if b.Output == nil {
// 		return errors.New("filesystem not open")
// 	}

// 	// Write the superblock
// 	sb := types.SuperBlockFromInfo(b.SuperBlock)

// 	// Update superblock fields
// 	sb.InodeCount = uint32(len(b.InodeMap))
// 	blocks := uint32(b.CurrentPos >> b.SuperBlock.BlksizeBits)
// 	if b.CurrentPos%(1<<b.SuperBlock.BlksizeBits) != 0 {
// 		blocks++
// 	}
// 	sb.Blocks = blocks

// 	// Set UUID if not already set
// 	for i, byt := range sb.UUIDBytes {
// 		if byt != 0 {
// 			break
// 		}
// 		if i == len(sb.UUIDBytes)-1 {
// 			// UUID is all zeros, generate a random one
// 			b.SuperBlock.SetRandomUUID()
// 			sb.UUIDBytes = b.SuperBlock.UUID
// 		}
// 	}

// 	// Calculate and set checksum
// 	if sb.HasFeatureCompat(uint16(types.EROFS_FEATURE_COMPAT_SB_CHKSUM)) {
// 		sb.SetChecksum()
// 	}

// 	// Seek to beginning of file
// 	_, err := b.Output.Seek(0, io.SeekStart)
// 	if err != nil {
// 		return err
// 	}

// 	// Write superblock
// 	err = binary.Write(b.Output, binary.LittleEndian, sb)
// 	if err != nil {
// 		return err
// 	}

// 	return b.Output.Close()
// }

func (b *Builder) Close() error {
	if b.Output == nil {
		return errors.New("filesystem not open")
	}

	// Write the superblock
	sb := types.SuperBlockFromInfo(b.SuperBlock)

	// Update superblock fields
	sb.InodeCount = uint64(len(b.InodeMap))
	blocks := uint64(b.CurrentPos >> b.SuperBlock.BlksizeBits)
	if b.CurrentPos%(1<<b.SuperBlock.BlksizeBits) != 0 {
		blocks++
	}
	sb.TotalBlocks = blocks

	// Set UUID if not already set
	for i, byt := range sb.UUIDBytes {
		if byt != 0 {
			break
		}
		if i == len(sb.UUIDBytes)-1 {
			// UUID is all zeros, generate a random one
			b.SuperBlock.SetRandomUUID()
			sb.UUIDBytes = b.SuperBlock.UUID
		}
	}

	// Debug output superblock values
	types.Erofs_info("Finalizing EROFS filesystem:")
	types.Erofs_info("Magic:            0x%08x", sb.Magic)
	types.Erofs_info("Block Size:       %d bytes (ilog2: %d)", 1<<sb.BlocksizeIlog, sb.BlocksizeIlog)
	types.Erofs_info("Feature Compat:   0x%04x", sb.FeatureCompat)
	types.Erofs_info("Feature Incompat: 0x%04x", sb.FeatureIncompat)
	types.Erofs_info("Inode Count:      %d", sb.InodeCount)
	types.Erofs_info("Blocks:           %d", sb.TotalBlocks)
	types.Erofs_info("Current Position: %d", b.CurrentPos)

	// Calculate and set checksum
	if sb.HasFeatureCompat(uint32(types.EROFS_FEATURE_COMPAT_SB_CHKSUM)) {
		sb.SetChecksum()
		types.Erofs_info("Checksum:         0x%08x", sb.Checksum)
	}

	// Debug output superblock structure
	buf := new(bytes.Buffer)
	err := binary.Write(buf, binary.LittleEndian, sb)
	if err != nil {
		return fmt.Errorf("error encoding superblock: %v", err)
	}
	sbBytes := buf.Bytes()
	types.Erofs_debug("Superblock binary representation:")
	types.DumpHex(sbBytes[:64], "SB") // First 64 bytes

	// Seek to beginning of file
	_, err = b.Output.Seek(0, io.SeekStart)
	if err != nil {
		return err
	}

	// Write superblock
	bytesWritten, err := b.Output.Write(sbBytes)
	if err != nil {
		return err
	}
	types.Erofs_info("Wrote %d bytes for superblock", bytesWritten)

	// Double-check the magic was written correctly
	testBuf := make([]byte, 4)
	b.Output.ReadAt(testBuf, 0)
	magic := binary.LittleEndian.Uint32(testBuf)
	types.Erofs_info("Verification of written magic: 0x%08x", magic)
	if magic != types.EROFS_SUPER_MAGIC_V1 {
		types.Erofs_err("Magic number mismatch after writing! Found 0x%08x, expected 0x%08x",
			magic, types.EROFS_SUPER_MAGIC_V1)
	}

	return b.Output.Close()
}
