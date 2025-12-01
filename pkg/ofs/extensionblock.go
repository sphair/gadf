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

	// Header
	adf.PutLong(block, adf.OffsetBlockType, adf.TypeList)
	adf.PutLong(block, adf.OffsetHeaderKey, eb.HeaderKey)
	adf.PutLong(block, adf.OffsetHighSeq, uint32(len(eb.DataBlocks)))

	// Data block pointers (stored in reverse order)
	for i, db := range eb.DataBlocks {
		adf.PutLong(block, adf.OffsetFirstDataPtr-i, db)
	}

	// Footer
	adf.PutLong(block, adf.OffsetExtension, eb.Extension)
	adf.PutLong(block, adf.OffsetSecondaryType, adf.STFile)

	adf.SetBlockChecksum(block, adf.OffsetChecksum)

	return block
}

// WriteBlock writes the extension block to disk
func (eb *ExtensionBlock) WriteBlock(disk *adf.Disk) error {
	block := eb.CreateBlock()
	return disk.WriteBlock(int(eb.BlockNum), block)
}
