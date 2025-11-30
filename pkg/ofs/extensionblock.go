package ofs

import (
	"github.com/sphair/gadf/pkg/adf"
)

// ExtensionBlock represents an OFS file extension block.
// Used when a file has more than 72 data blocks (> ~35KB).
// Each extension block can hold up to 72 additional data block pointers.
type ExtensionBlock struct {
	BlockNum   uint32   // This block's number
	HeaderKey  uint32   // File header block number (parent)
	DataBlocks []uint32 // Data block pointers (max 72)
	Extension  uint32   // Next extension block (0 if none)
}

// NewExtensionBlock creates a new file extension block
func NewExtensionBlock(blockNum, headerKey uint32) *ExtensionBlock {
	return &ExtensionBlock{
		BlockNum:   blockNum,
		HeaderKey:  headerKey,
		DataBlocks: make([]uint32, 0, MaxDataBlocks),
		Extension:  0,
	}
}

// AddDataBlock adds a data block pointer to the extension block
// Returns false if the extension block is full (need another extension block)
func (eb *ExtensionBlock) AddDataBlock(blockNum uint32) bool {
	if len(eb.DataBlocks) >= MaxDataBlocks {
		return false
	}
	eb.DataBlocks = append(eb.DataBlocks, blockNum)
	return true
}

// CreateBlock creates the raw block data for the extension block
func (eb *ExtensionBlock) CreateBlock() []byte {
	block := make([]byte, adf.BlockSize)

	// Type at offset 0: T_LIST (16)
	adf.PutLong(block, 0, adf.TypeList)

	// Header key (file header pointer) at offset 4
	adf.PutLong(block, 1, eb.HeaderKey)

	// High seq (number of data blocks in this extension) at offset 8
	adf.PutLong(block, 2, uint32(len(eb.DataBlocks)))

	// Data blocks table: stored in reverse order starting at offset -51
	for i, db := range eb.DataBlocks {
		adf.PutLong(block, -51-i, db)
	}

	// Extension at offset -2 (next extension block, or 0)
	adf.PutLong(block, -2, eb.Extension)

	// Secondary type at offset -1: ST_FILE (-3 as uint32)
	adf.PutLong(block, -1, adf.STFile)

	// Calculate and set checksum at offset 5
	adf.SetBlockChecksum(block, 5)

	return block
}

// WriteBlock writes the extension block to disk
func (eb *ExtensionBlock) WriteBlock(disk *adf.Disk) error {
	block := eb.CreateBlock()
	return disk.WriteBlock(int(eb.BlockNum), block)
}

