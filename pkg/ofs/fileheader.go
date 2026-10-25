package ofs

import (
	"time"

	"github.com/sphair/gadf/pkg/adf"
)

// MaxDataBlocks is the maximum number of data block pointers in a file header
// (BlockSize/4) - 56 = 72 for 512-byte blocks
const MaxDataBlocks = 72

// FileHeader represents an OFS file header block
type FileHeader struct {
	BlockNum   uint32   // This block's number
	Parent     uint32   // Parent directory block number
	Name       string   // Filename (max 30 chars)
	FileSize   uint32   // File size in bytes
	DataBlocks []uint32 // Data block pointers (max 72)
	Extension  uint32   // First file extension block (0 if none)
	HashChain  uint32   // Next entry with same hash (0 if none)
	ModifyTime time.Time
	Protection uint32 // Protection flags (0 = default)

	extensions []*ExtensionBlock // owned extension blocks (written by WriteBlock)
}

// NewFileHeader creates a new file header
func NewFileHeader(blockNum, parent uint32, name string, fileSize uint32) *FileHeader {
	return &FileHeader{
		BlockNum:   blockNum,
		Parent:     parent,
		Name:       name,
		FileSize:   fileSize,
		DataBlocks: make([]uint32, 0, MaxDataBlocks),
		ModifyTime: time.Now(),
		Protection: 0,
	}
}

// RequiredExtensionBlocks returns how many extension blocks are needed for n data blocks
func RequiredExtensionBlocks(numDataBlocks int) int {
	if numDataBlocks <= MaxDataBlocks {
		return 0
	}
	remaining := numDataBlocks - MaxDataBlocks
	return (remaining + MaxDataBlocks - 1) / MaxDataBlocks
}

// AssignDataBlocks distributes data block pointers across file header and extension blocks.
// extBlockNums should have exactly RequiredExtensionBlocks(len(dataBlockNums)) entries.
// Extension blocks are stored internally and written by WriteBlock.
func (fh *FileHeader) AssignDataBlocks(dataBlockNums []uint32, extBlockNums []uint32) {
	idx := 0

	// Fill file header (first 72 data blocks)
	for idx < len(dataBlockNums) && idx < MaxDataBlocks {
		fh.DataBlocks = append(fh.DataBlocks, dataBlockNums[idx])
		idx++
	}

	// Create extension blocks for the rest
	for i := range extBlockNums {
		eb := NewExtensionBlock(extBlockNums[i], fh.BlockNum)
		for j := 0; j < MaxDataBlocks && idx < len(dataBlockNums); j++ {
			eb.AddDataBlock(dataBlockNums[idx])
			idx++
		}
		fh.extensions = append(fh.extensions, eb)
	}

	// Chain extension blocks
	if len(fh.extensions) > 0 {
		fh.Extension = fh.extensions[0].BlockNum
		for i := 0; i < len(fh.extensions)-1; i++ {
			fh.extensions[i].Extension = fh.extensions[i+1].BlockNum
		}
	}
}

// CreateBlock creates the raw block data for the file header
func (fh *FileHeader) CreateBlock() []byte {
	block := make([]byte, adf.BlockSize)

	// Header
	adf.PutLong(block, adf.OffsetBlockType, adf.TypeHeader)
	adf.PutLong(block, adf.OffsetHeaderKey, fh.BlockNum)
	adf.PutLong(block, adf.OffsetHighSeq, uint32(len(fh.DataBlocks)))
	adf.PutLong(block, adf.OffsetHashTableSize, 0)
	if len(fh.DataBlocks) > 0 {
		adf.PutLong(block, adf.OffsetFirstDataLong, fh.DataBlocks[0])
	}

	// Data block pointers (stored in reverse order)
	for i, db := range fh.DataBlocks {
		adf.PutLong(block, adf.OffsetFirstDataPtr-i, db)
	}

	// File metadata
	adf.PutLong(block, adf.OffsetProtection, fh.Protection)
	adf.PutLong(block, adf.OffsetFileSize, fh.FileSize)

	days, mins, ticks := timeToAmiga(fh.ModifyTime)
	adf.PutLong(block, adf.OffsetDateDays, days)
	adf.PutLong(block, adf.OffsetDateMins, mins)
	adf.PutLong(block, adf.OffsetDateTicks, ticks)

	// Footer
	adf.PutBStr(block, adf.OffsetName, adf.MaxFilenameLen, fh.Name)
	adf.PutLong(block, adf.OffsetNextHash, fh.HashChain)
	adf.PutLong(block, adf.OffsetParent, fh.Parent)
	adf.PutLong(block, adf.OffsetExtension, fh.Extension)
	adf.PutLong(block, adf.OffsetSecondaryType, adf.STFile)

	adf.SetBlockChecksum(block, adf.OffsetChecksum)

	return block
}

// WriteBlock writes the file header and all extension blocks to disk
func (fh *FileHeader) WriteBlock(disk *adf.Disk) error {
	block := fh.CreateBlock()
	if err := disk.WriteBlock(int(fh.BlockNum), block); err != nil {
		return err
	}

	for _, eb := range fh.extensions {
		if err := eb.WriteBlock(disk); err != nil {
			return err
		}
	}
	return nil
}
