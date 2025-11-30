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
	BlockNum    uint32     // This block's number
	Parent      uint32     // Parent directory block number
	Name        string     // Filename (max 30 chars)
	FileSize    uint32     // File size in bytes
	DataBlocks  []uint32   // Data block pointers (max 72)
	Extension   uint32     // First file extension block (0 if none)
	HashChain   uint32     // Next entry with same hash (0 if none)
	ModifyTime  time.Time
	Protection  uint32     // Protection flags (0 = default)

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
	adf.PutLong(block, 0, adf.TypeHeader)

	// Header key (self pointer) at offset 4
	adf.PutLong(block, 1, fh.BlockNum)

	// High seq (number of data blocks in this header) at offset 8
	adf.PutLong(block, 2, uint32(len(fh.DataBlocks)))

	// Data size at offset 12: unused (0)
	adf.PutLong(block, 3, 0)

	// First data at offset 16
	if len(fh.DataBlocks) > 0 {
		adf.PutLong(block, 4, fh.DataBlocks[0])
	}

	// Checksum at offset 20: calculated later
	// adf.PutLong(block, 5, 0)

	// Data blocks table at offset 24
	// Data blocks are stored in reverse order: first block at offset -51, last at -51-(n-1)
	for i, db := range fh.DataBlocks {
		adf.PutLong(block, -51-i, db)
	}

	// Protection at offset -48
	adf.PutLong(block, -48, fh.Protection)

	// Byte size at offset -47
	adf.PutLong(block, -47, fh.FileSize)

	// Comment at offset -46 (BCPL string, max 79 chars) - empty
	// adf.PutBStr(block, -46, 79, "")

	// Modify date at offset -23 to -21
	days, mins, ticks := timeToAmiga(fh.ModifyTime)
	adf.PutLong(block, -23, days)
	adf.PutLong(block, -22, mins)
	adf.PutLong(block, -21, ticks)

	// Name
	adf.PutBStr(block, adf.OffsetName, adf.MaxFilenameLen, fh.Name)

	// Hash chain at offset -4
	adf.PutLong(block, -4, fh.HashChain)

	// Parent at offset -3
	adf.PutLong(block, -3, fh.Parent)

	// Extension at offset -2
	adf.PutLong(block, -2, fh.Extension)

	// Secondary type at offset -1: ST_FILE (-3 as uint32)
	adf.PutLong(block, -1, adf.STFile)

	// Calculate and set checksum at offset 5
	adf.SetBlockChecksum(block, 5)

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

