// Package ofs provides OFS (Old File System) structures for AmigaDOS 1.2+
package ofs

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/sphair/gadf/pkg/adf"
)

// Builder creates bootable ADF disk images
type Builder struct {
	disk        *adf.Disk
	bitmap      *Bitmap
	rootBlock   *RootBlock
	label       string
	minimalBoot bool // Use minimal bootblock (no expansion.library)

	// Track directories we've created (normalized paths using '/' separators)
	directories map[string]*DirBlock // path -> DirBlock

	// Track all files added (full path on disk)
	files []string
}

// NewBuilder creates a new ADF builder
func NewBuilder(label string) *Builder {
	return &Builder{
		disk:        adf.NewDisk(),
		bitmap:      NewBitmap(),
		label:       label,
		minimalBoot: false,
		directories: make(map[string]*DirBlock),
		files:       []string{},
	}
}

func (b *Builder) SetMinimalBoot(minimal bool) {
	b.minimalBoot = minimal
}

// ValidateFilename returns error if name contains : or / or exceeds 30 chars
func ValidateFilename(name string) error {
	if len(name) == 0 {
		return fmt.Errorf("filename cannot be empty")
	}
	if len(name) > adf.MaxFilenameLen {
		return fmt.Errorf("filename too long: %d chars (max %d)", len(name), adf.MaxFilenameLen)
	}
	if strings.ContainsAny(name, ":/") {
		return fmt.Errorf("filename contains invalid characters (: or /): %s", name)
	}
	return nil
}

// normalizePath converts any host-specific separators into Amiga-style paths
// and removes duplicate/leading/trailing separators.
func normalizePath(path string) string {
	path = strings.ReplaceAll(path, "\\", "/")
	parts := strings.Split(path, "/")
	filtered := make([]string, 0, len(parts))
	for _, part := range parts {
		if part != "" {
			filtered = append(filtered, part)
		}
	}
	return strings.Join(filtered, "/")
}

// joinAmigaPath joins path segments using '/' regardless of host OS.
func joinAmigaPath(parent, child string) string {
	parent = normalizePath(parent)
	child = normalizePath(child)
	switch {
	case parent == "":
		return child
	case child == "":
		return parent
	default:
		return parent + "/" + child
	}
}

func (b *Builder) Build() error {
	// Reserve system blocks
	b.bitmap.AllocateSpecific(adf.RootBlock)
	b.bitmap.AllocateSpecific(adf.BitmapBlock)

	if err := WriteBootBlock(b.disk, b.minimalBoot); err != nil {
		return fmt.Errorf("failed to write bootblock: %w", err)
	}

	b.rootBlock = NewRootBlock(b.label, adf.BitmapBlock)
	return nil
}

// allocateBlocks allocates n blocks and returns their block numbers
func (b *Builder) allocateBlocks(n int) []uint32 {
	blocks := make([]uint32, n)
	for i := range n {
		blocks[i] = uint32(b.bitmap.AllocateNext())
	}
	return blocks
}

func (b *Builder) Finalize() error {
	if err := b.rootBlock.WriteBlock(b.disk); err != nil {
		return fmt.Errorf("failed to write rootblock: %w", err)
	}

	if err := b.bitmap.WriteBitmapBlock(b.disk, adf.BitmapBlock); err != nil {
		return fmt.Errorf("failed to write bitmap: %w", err)
	}

	return nil
}

// Save writes the disk image to a file
func (b *Builder) Save(filename string) error {
	return b.disk.Save(filename)
}

// AddFile adds a file to the disk in the specified directory.
// dirPath is "" for root, or a path like "S" or "Data/Levels".
// Supports files up to disk capacity using extension blocks.
func (b *Builder) AddFile(dirPath, name string, data []byte) error {
	if err := ValidateFilename(name); err != nil {
		return err
	}

	dirPath = normalizePath(dirPath)

	var parentBlock uint32
	var parentDir *DirBlock

	if dirPath == "" {
		parentBlock = adf.RootBlock
	} else {
		dir, err := b.getOrCreateDir(dirPath)
		if err != nil {
			return err
		}
		parentBlock = dir.BlockNum
		parentDir = dir
	}

	var chunks [][]byte
	if len(data) > 0 {
		chunks = SplitFileData(data)
	}
	numExtBlocks := RequiredExtensionBlocks(len(chunks))

	totalBlocksNeeded := 1 + numExtBlocks + len(chunks)
	if b.bitmap.CountFree() < totalBlocksNeeded {
		return fmt.Errorf("disk full: need %d blocks for %s, only %d free", totalBlocksNeeded, name, b.bitmap.CountFree())
	}

	headerBlockNum := b.bitmap.AllocateNext()
	if headerBlockNum < 0 {
		return fmt.Errorf("disk full: cannot allocate file header for %s", name)
	}

	extBlockNums := b.allocateBlocks(numExtBlocks)
	dataBlockNums := b.allocateBlocks(len(chunks))

	// Create file header
	fh := NewFileHeader(uint32(headerBlockNum), parentBlock, name, uint32(len(data)))
	fh.AssignDataBlocks(dataBlockNums, extBlockNums)
	fh.HashChain = b.addHashEntry(parentDir, name, uint32(headerBlockNum))

	if err := fh.WriteBlock(b.disk); err != nil {
		return fmt.Errorf("failed to write file header for %s: %w", name, err)
	}

	// Write data blocks
	for i, chunk := range chunks {
		db := NewDataBlock(dataBlockNums[i], uint32(headerBlockNum), uint32(i+1), chunk)
		if i < len(chunks)-1 {
			db.NextData = dataBlockNums[i+1]
		}
		if err := db.WriteBlock(b.disk); err != nil {
			return fmt.Errorf("failed to write data block %d for %s: %w", i, name, err)
		}
	}

	// Parent directory must be rewritten (root block is written in Finalize)
	if parentDir != nil {
		if err := parentDir.WriteBlock(b.disk); err != nil {
			return fmt.Errorf("failed to update directory %s: %w", dirPath, err)
		}
	}

	if dirPath == "" {
		b.files = append(b.files, name)
	} else {
		b.files = append(b.files, joinAmigaPath(dirPath, name))
	}

	return nil
}

func (b *Builder) AddFileFromDisk(dirPath, diskPath string) error {
	data, err := os.ReadFile(diskPath)
	if err != nil {
		return fmt.Errorf("failed to read file %s: %w", diskPath, err)
	}

	name := filepath.Base(diskPath)
	return b.AddFile(dirPath, name, data)
}

// getOrCreateDir gets or creates a directory at the specified path
func (b *Builder) getOrCreateDir(path string) (*DirBlock, error) {
	normPath := normalizePath(path)
	if normPath == "" {
		return nil, nil
	}

	if dir, ok := b.directories[normPath]; ok {
		return dir, nil
	}

	parts := strings.Split(normPath, "/")
	var parentDir *DirBlock
	parentBlock := uint32(adf.RootBlock)
	currentPath := ""

	for _, part := range parts {
		if err := ValidateFilename(part); err != nil {
			return nil, fmt.Errorf("invalid directory name: %w", err)
		}

		currentPath = joinAmigaPath(currentPath, part)

		if dir, ok := b.directories[currentPath]; ok {
			parentDir = dir
			parentBlock = dir.BlockNum
			continue
		}

		blockNum := b.bitmap.AllocateNext()
		if blockNum < 0 {
			return nil, fmt.Errorf("disk full: cannot allocate directory block for %s", currentPath)
		}

		dir := NewDirBlock(uint32(blockNum), parentBlock, part)
		dir.HashChain = b.addHashEntry(parentDir, part, uint32(blockNum))

		if err := dir.WriteBlock(b.disk); err != nil {
			return nil, fmt.Errorf("failed to write directory block for %s: %w", currentPath, err)
		}

		b.directories[currentPath] = dir
		parentDir = dir
		parentBlock = dir.BlockNum
	}

	return b.directories[normPath], nil
}

// CreateStartupSequence creates the S/startup-sequence file
func (b *Builder) CreateStartupSequence(executableName string) error {
	if _, err := b.getOrCreateDir("S"); err != nil {
		return err
	}

	content := executableName + "\n"
	return b.AddFile("S", "startup-sequence", []byte(content))
}

// AddDirectory adds all files from a directory recursively.
// If flatten is true, all files are placed at destPath (flattening the structure).
// If flatten is false, the directory structure is preserved.
// Returns a list of added items (files and directories) for reporting.
func (b *Builder) AddDirectory(sourcePath, destPath string, flatten bool) ([]string, error) {
	var added []string

	destPath = normalizePath(destPath)

	if flatten {
		err := filepath.Walk(sourcePath, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}

			if info.IsDir() {
				return nil
			}

			if err := b.AddFileFromDisk(destPath, path); err != nil {
				return err
			}

			baseName := filepath.Base(path)
			if destPath == "" {
				added = append(added, baseName)
			} else {
				added = append(added, joinAmigaPath(destPath, baseName))
			}

			return nil
		})

		if err != nil {
			return added, fmt.Errorf("failed to walk directory %s: %w", sourcePath, err)
		}
	} else {
		subAdded, err := b.addDirectoryRecursive(sourcePath, destPath)
		if err != nil {
			return nil, err
		}
		added = append(added, subAdded...)
	}

	return added, nil
}

// addDirectoryRecursive is the internal recursive implementation
func (b *Builder) addDirectoryRecursive(sourcePath, destPath string) ([]string, error) {
	var added []string

	entries, err := os.ReadDir(sourcePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read directory %s: %w", sourcePath, err)
	}

	for _, entry := range entries {
		srcFilePath := filepath.Join(sourcePath, entry.Name())

		if entry.IsDir() {
			subDestPath := joinAmigaPath(destPath, entry.Name())
			if _, err := b.getOrCreateDir(subDestPath); err != nil {
				return added, fmt.Errorf("failed to create directory %s: %w", subDestPath, err)
			}
			added = append(added, subDestPath+"/")

			subAdded, err := b.addDirectoryRecursive(srcFilePath, subDestPath)
			if err != nil {
				return added, err
			}
			added = append(added, subAdded...)
			continue
		}

		if err := b.AddFileFromDisk(destPath, srcFilePath); err != nil {
			return added, err
		}
		if destPath == "" {
			added = append(added, entry.Name())
		} else {
			added = append(added, joinAmigaPath(destPath, entry.Name()))
		}
	}

	return added, nil
}

func (b *Builder) addHashEntry(parentDir *DirBlock, name string, blockNum uint32) uint32 {
	if parentDir != nil {
		return parentDir.AddHashEntry(name, blockNum)
	}
	return b.rootBlock.AddHashEntry(name, blockNum)
}

// GetFreeBlocks returns the number of free blocks remaining
func (b *Builder) GetFreeBlocks() int {
	return b.bitmap.CountFree()
}

// GetFileList returns a sorted list of all files on the disk
func (b *Builder) GetFileList() []string {
	return b.files
}

// GetDirectories returns a sorted list of all directories on the disk
func (b *Builder) GetDirectories() []string {
	dirs := make([]string, 0, len(b.directories))
	for dir := range b.directories {
		dirs = append(dirs, dir)
	}
	return dirs
}
