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

// AddDataBlock adds a data block pointer to the file header
// Returns false if the file header is full (need extension block)
func (fh *FileHeader) AddDataBlock(blockNum uint32) bool {
	if len(fh.DataBlocks) >= MaxDataBlocks {
		return false
	}
	fh.DataBlocks = append(fh.DataBlocks, blockNum)
	return true
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
	for i := 0; i < len(extBlockNums); i++ {
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

	// Type at offset 0: T_HEADER (2)
	adf.PutLong(block, adf.OffsetBlockType, adf.TypeHeader)

	// Header key (self pointer) at offset 4
	adf.PutLong(block, adf.OffsetHeaderKey, fh.BlockNum)

	// High seq (number of data blocks in this header) at offset 8
	adf.PutLong(block, adf.OffsetHighSeq, uint32(len(fh.DataBlocks)))

	// Data size at offset 12: unused (0)
	adf.PutLong(block, adf.OffsetHashTableSize, 0)

	// First data at offset 16
	if len(fh.DataBlocks) > 0 {
		adf.PutLong(block, adf.OffsetFirstDataLong, fh.DataBlocks[0])
	}

	// Checksum at offset 20: calculated later
	// adf.PutLong(block, 5, 0)

	// Data blocks table at offset 24
	// Data blocks are stored in reverse order: first block at OffsetFirstDataPtr
	for i, db := range fh.DataBlocks {
		adf.PutLong(block, adf.OffsetFirstDataPtr-i, db)
	}

	// Protection
	adf.PutLong(block, adf.OffsetProtection, fh.Protection)

	// Byte size
	adf.PutLong(block, adf.OffsetFileSize, fh.FileSize)

	// Comment at offset -46 (BCPL string, max 79 chars) - empty
	// adf.PutBStr(block, -46, 79, "")

	// Modify date
	days, mins, ticks := timeToAmiga(fh.ModifyTime)
	adf.PutLong(block, adf.OffsetDateDays, days)
	adf.PutLong(block, adf.OffsetDateMins, mins)
	adf.PutLong(block, adf.OffsetDateTicks, ticks)

	// Name
	adf.PutBStr(block, adf.OffsetName, adf.MaxFilenameLen, fh.Name)

	// Hash chain pointer
	adf.PutLong(block, adf.OffsetNextHash, fh.HashChain)

	// Parent pointer
	adf.PutLong(block, adf.OffsetParent, fh.Parent)

	// Extension pointer
	adf.PutLong(block, adf.OffsetExtension, fh.Extension)

	// Secondary type: ST_FILE (-3)
	adf.PutLong(block, adf.OffsetSecondaryType, adf.STFile)

	// Calculate and set checksum
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
