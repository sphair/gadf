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

	// Track directories we've created
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

func (b *Builder) Build() error {
	// Reserve the rootblock and bitmap block
	b.bitmap.AllocateSpecific(adf.RootBlock)
	b.bitmap.AllocateSpecific(adf.BitmapBlock)

	// Create and write bootblock
	if err := WriteBootBlock(b.disk, b.minimalBoot); err != nil {
		return fmt.Errorf("failed to write bootblock: %w", err)
	}

	// Create root block
	b.rootBlock = NewRootBlock(b.label, adf.BitmapBlock)

	return nil
}

// allocateBlocks allocates n blocks and returns their block numbers
func (b *Builder) allocateBlocks(n int) []uint32 {
	blocks := make([]uint32, n)
	for i := 0; i < n; i++ {
		blocks[i] = uint32(b.bitmap.AllocateNext())
	}
	return blocks
}

func (b *Builder) Finalize() error {
	// Write root block
	if err := b.rootBlock.WriteBlock(b.disk); err != nil {
		return fmt.Errorf("failed to write rootblock: %w", err)
	}

	// Write bitmap
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

	// Get or create the parent directory
	var parentBlock uint32
	var parentDir *DirBlock // nil for root

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

	// Split file data into chunks
	chunks := SplitFileData(data)
	numExtBlocks := RequiredExtensionBlocks(len(chunks))

	// Check if we have enough space: 1 file header + extension blocks + data blocks
	totalBlocksNeeded := 1 + numExtBlocks + len(chunks)
	if b.bitmap.CountFree() < totalBlocksNeeded {
		return fmt.Errorf("disk full: need %d blocks for %s, only %d free", totalBlocksNeeded, name, b.bitmap.CountFree())
	}

	// Allocate all blocks
	headerBlockNum := b.bitmap.AllocateNext()
	if headerBlockNum < 0 {
		return fmt.Errorf("disk full: cannot allocate file header for %s", name)
	}

	extBlockNums := b.allocateBlocks(numExtBlocks)
	dataBlockNums := b.allocateBlocks(len(chunks))

	// Create file header and assign data blocks (extension blocks stored internally)
	fh := NewFileHeader(uint32(headerBlockNum), parentBlock, name, uint32(len(data)))
	fh.AssignDataBlocks(dataBlockNums, extBlockNums)

	// Add to parent's hash table
	if parentDir != nil {
		fh.HashChain = parentDir.AddHashEntry(name, uint32(headerBlockNum))
	} else {
		fh.HashChain = b.rootBlock.AddHashEntry(name, uint32(headerBlockNum))
	}

	// Write file header and extension blocks
	if err := fh.WriteBlock(b.disk); err != nil {
		return fmt.Errorf("failed to write file header for %s: %w", name, err)
	}

	// Create and write data blocks
	for i, chunk := range chunks {
		db := NewDataBlock(dataBlockNums[i], uint32(headerBlockNum), uint32(i+1), chunk)
		if i < len(chunks)-1 {
			db.NextData = dataBlockNums[i+1]
		}
		if err := db.WriteBlock(b.disk); err != nil {
			return fmt.Errorf("failed to write data block %d for %s: %w", i, name, err)
		}
	}

	// Rewrite parent directory block if it's not the root
	// (root block is written in Finalize)
	if parentDir != nil {
		if err := parentDir.WriteBlock(b.disk); err != nil {
			return fmt.Errorf("failed to update directory %s: %w", dirPath, err)
		}
	}

	// Track the file
	if dirPath == "" {
		b.files = append(b.files, name)
	} else {
		b.files = append(b.files, filepath.Join(dirPath, name))
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
	// Check if already exists
	if dir, ok := b.directories[path]; ok {
		return dir, nil
	}

	// Validate directory name
	if err := ValidateFilename(path); err != nil {
		return nil, fmt.Errorf("invalid directory name: %w", err)
	}

	// Allocate directory block
	blockNum := b.bitmap.AllocateNext()
	if blockNum < 0 {
		return nil, fmt.Errorf("disk full: cannot allocate directory block for %s", path)
	}

	// Create directory block
	dir := NewDirBlock(uint32(blockNum), adf.RootBlock, path)

	// Add to root's hash table
	dir.HashChain = b.rootBlock.AddHashEntry(path, uint32(blockNum))

	// Write directory block
	if err := dir.WriteBlock(b.disk); err != nil {
		return nil, fmt.Errorf("failed to write directory block for %s: %w", path, err)
	}

	b.directories[path] = dir
	return dir, nil
}

// CreateStartupSequence creates the S/startup-sequence file
func (b *Builder) CreateStartupSequence(executableName string) error {
	// Ensure S directory exists
	if _, err := b.getOrCreateDir("S"); err != nil {
		return err
	}

	// Create startup-sequence content
	content := executableName + "\n"

	return b.AddFile("S", "startup-sequence", []byte(content))
}

// AddDirectory adds all files from a directory recursively.
// If flatten is true, all files are placed at destPath (flattening the structure).
// If flatten is false, the directory structure is preserved.
// Returns a list of added items (files and directories) for reporting.
func (b *Builder) AddDirectory(sourcePath, destPath string, flatten bool) ([]string, error) {
	var added []string

	if flatten {
		// Walk all files recursively and add them to the destination path
		err := filepath.Walk(sourcePath, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}

			// Skip directories, only process files
			if info.IsDir() {
				return nil
			}

			// Add file to the destination path (flattened)
			if err := b.AddFileFromDisk(destPath, path); err != nil {
				return err
			}

			// Track what was added
			if destPath == "" {
				added = append(added, filepath.Base(path))
			} else {
				added = append(added, filepath.Join(destPath, filepath.Base(path)))
			}

			return nil
		})

		if err != nil {
			return added, fmt.Errorf("failed to walk directory %s: %w", sourcePath, err)
		}
	} else {
		// Recursively add files preserving directory structure
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
			// Create the subdirectory on disk
			subDestPath := filepath.Join(destPath, entry.Name())
			if _, err := b.getOrCreateDir(subDestPath); err != nil {
				return added, fmt.Errorf("failed to create directory %s: %w", subDestPath, err)
			}
			added = append(added, subDestPath+"/")

			// Recurse into it
			subAdded, err := b.addDirectoryRecursive(srcFilePath, subDestPath)
			if err != nil {
				return added, err
			}
			added = append(added, subAdded...)
			continue
		}

		// Add file to destination directory
		if err := b.AddFileFromDisk(destPath, srcFilePath); err != nil {
			return added, err
		}
		if destPath == "" {
			added = append(added, entry.Name())
		} else {
			added = append(added, filepath.Join(destPath, entry.Name()))
		}
	}

	return added, nil
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

