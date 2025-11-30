package ofs

import (
	"github.com/sphair/gadf/pkg/adf"
)

// OFS data block constants
const (
	// OFSDataSize = 512 - 24 byte header = 488 bytes per block.
	// FFS would give us full 512 but we target KS 1.2 compatibility.
	OFSDataSize = 488
)

// DataBlock represents an OFS data block
type DataBlock struct {
	BlockNum  uint32 // This block's number
	HeaderKey uint32 // File header block number
	SeqNum    uint32 // Sequence number (1-based)
	DataSize  uint32 // Actual data size in this block
	NextData  uint32 // Next data block (0 if last)
	Data      []byte // The actual file data (max 488 bytes)
}

// NewDataBlock creates a new OFS data block
func NewDataBlock(blockNum, headerKey, seqNum uint32, data []byte) *DataBlock {
	db := &DataBlock{
		BlockNum:  blockNum,
		HeaderKey: headerKey,
		SeqNum:    seqNum,
		DataSize:  uint32(len(data)),
		NextData:  0,
		Data:      make([]byte, len(data)),
	}
	copy(db.Data, data)
	return db
}

// CreateBlock creates the raw block data for the data block
func (db *DataBlock) CreateBlock() []byte {
	block := make([]byte, adf.BlockSize)

	// Type at offset 0: T_DATA (8)
	adf.PutLong(block, 0, adf.TypeData)

	// Header key at offset 4
	adf.PutLong(block, 1, db.HeaderKey)

	// Sequence number at offset 8
	adf.PutLong(block, 2, db.SeqNum)

	// Data size at offset 12
	adf.PutLong(block, 3, db.DataSize)

	// Next data at offset 16
	adf.PutLong(block, 4, db.NextData)

	// Checksum at offset 20: calculated later
	// adf.PutLong(block, 5, 0)

	// Data at offset 24
	copy(block[24:], db.Data)

	// Calculate and set checksum at offset 5
	adf.SetBlockChecksum(block, 5)

	return block
}

// WriteBlock writes the data block to disk
func (db *DataBlock) WriteBlock(disk *adf.Disk) error {
	block := db.CreateBlock()
	return disk.WriteBlock(int(db.BlockNum), block)
}

// SplitFileData splits file data into OFS data block chunks
func SplitFileData(data []byte) [][]byte {
	var chunks [][]byte
	for i := 0; i < len(data); i += OFSDataSize {
		end := i + OFSDataSize
		if end > len(data) {
			end = len(data)
		}
		chunk := make([]byte, end-i)
		copy(chunk, data[i:end])
		chunks = append(chunks, chunk)
	}
	// If empty file, still return one empty chunk
	if len(chunks) == 0 {
		chunks = append(chunks, []byte{})
	}
	return chunks
}

