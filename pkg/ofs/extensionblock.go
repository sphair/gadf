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
	adf.PutLong(block, adf.OffsetBlockType, adf.TypeList)

	// Header key (file header pointer) at offset 4
	adf.PutLong(block, adf.OffsetHeaderKey, eb.HeaderKey)

	// High seq (number of data blocks in this extension) at offset 8
	adf.PutLong(block, adf.OffsetHighSeq, uint32(len(eb.DataBlocks)))

	// Data blocks table: stored in reverse order starting at OffsetFirstDataPtr
	for i, db := range eb.DataBlocks {
		adf.PutLong(block, adf.OffsetFirstDataPtr-i, db)
	}

	// Extension pointer (next extension block, or 0)
	adf.PutLong(block, adf.OffsetExtension, eb.Extension)

	// Secondary type: ST_FILE (-3)
	adf.PutLong(block, adf.OffsetSecondaryType, adf.STFile)

	// Calculate and set checksum
	adf.SetBlockChecksum(block, adf.OffsetChecksum)

	return block
}

// WriteBlock writes the extension block to disk
func (eb *ExtensionBlock) WriteBlock(disk *adf.Disk) error {
	block := eb.CreateBlock()
	return disk.WriteBlock(int(eb.BlockNum), block)
}
