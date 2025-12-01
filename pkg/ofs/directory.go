package ofs

import (
	"time"

	"github.com/sphair/gadf/pkg/adf"
)

// DirBlock represents an OFS directory block
type DirBlock struct {
	BlockNum   uint32     // This block's number
	Parent     uint32     // Parent directory block number
	Name       string     // Directory name (max 30 chars)
	HashTable  [72]uint32 // Hash table entries
	HashChain  uint32     // Next entry with same hash in parent (0 if none)
	ModifyTime time.Time
	Protection uint32 // Protection flags
}

// NewDirBlock creates a new directory block
func NewDirBlock(blockNum, parent uint32, name string) *DirBlock {
	return &DirBlock{
		BlockNum:   blockNum,
		Parent:     parent,
		Name:       name,
		ModifyTime: time.Now(),
		Protection: 0,
	}
}

// SetHashEntry sets a hash table entry
func (db *DirBlock) SetHashEntry(index int, blockNum uint32) {
	if index >= 0 && index < HashTableSize {
		db.HashTable[index] = blockNum
	}
}

// GetHashEntry gets a hash table entry
func (db *DirBlock) GetHashEntry(index int) uint32 {
	if index >= 0 && index < HashTableSize {
		return db.HashTable[index]
	}
	return 0
}

// AddHashEntry adds an entry to the hash table and returns the previous entry (for chaining).
// The caller should set the returned value as the new entry's HashChain.
func (db *DirBlock) AddHashEntry(name string, blockNum uint32) uint32 {
	index := HashName(name)
	previous := db.HashTable[index]
	db.HashTable[index] = blockNum
	return previous
}

// CreateBlock creates the raw block data for the directory block
func (db *DirBlock) CreateBlock() []byte {
	block := make([]byte, adf.BlockSize)

	// Type at offset 0: T_HEADER (2)
	adf.PutLong(block, adf.OffsetBlockType, adf.TypeHeader)

	// Header key (self pointer) at offset 4
	adf.PutLong(block, adf.OffsetHeaderKey, db.BlockNum)

	// Unused at offsets 8, 12, 16
	adf.PutLong(block, adf.OffsetHighSeq, 0)
	adf.PutLong(block, adf.OffsetHashTableSize, 0)
	adf.PutLong(block, adf.OffsetFirstDataLong, 0)

	// Checksum at offset 20: calculated later

	// Hash table at offset 24 (72 entries)
	for i := 0; i < HashTableSize; i++ {
		adf.PutLong(block, adf.OffsetHashTable+i, db.HashTable[i])
	}

	// Protection bits
	adf.PutLong(block, adf.OffsetProtection, db.Protection)

	// Modify date
	days, mins, ticks := timeToAmiga(db.ModifyTime)
	adf.PutLong(block, adf.OffsetDateDays, days)
	adf.PutLong(block, adf.OffsetDateMins, mins)
	adf.PutLong(block, adf.OffsetDateTicks, ticks)

	// Name
	adf.PutBStr(block, adf.OffsetName, adf.MaxFilenameLen, db.Name)

	// Hash chain pointer
	adf.PutLong(block, adf.OffsetNextHash, db.HashChain)

	// Parent pointer
	adf.PutLong(block, adf.OffsetParent, db.Parent)

	// Extension (unused for OFS)
	adf.PutLong(block, adf.OffsetExtension, 0)

	// Secondary type: ST_USERDIR
	adf.PutLong(block, adf.OffsetSecondaryType, adf.STUserDir)

	// Calculate and set checksum at offset 5
	adf.SetBlockChecksum(block, adf.OffsetChecksum)

	return block
}

// WriteBlock writes the directory block to disk
func (db *DirBlock) WriteBlock(disk *adf.Disk) error {
	block := db.CreateBlock()
	return disk.WriteBlock(int(db.BlockNum), block)
}
