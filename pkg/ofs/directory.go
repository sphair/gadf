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
	adf.PutLong(block, 0, adf.TypeHeader)

	// Header key (self pointer) at offset 4
	adf.PutLong(block, 1, db.BlockNum)

	// Unused at offsets 8, 12, 16
	adf.PutLong(block, 2, 0)
	adf.PutLong(block, 3, 0)
	adf.PutLong(block, 4, 0)

	// Checksum at offset 20: calculated later

	// Hash table at offset 24 (72 entries)
	for i := 0; i < HashTableSize; i++ {
		adf.PutLong(block, 6+i, db.HashTable[i])
	}

	// Protection at offset -48
	adf.PutLong(block, -48, db.Protection)

	// Modify date at offset -23 to -21
	days, mins, ticks := timeToAmiga(db.ModifyTime)
	adf.PutLong(block, -23, days)
	adf.PutLong(block, -22, mins)
	adf.PutLong(block, -21, ticks)

	// Name
	adf.PutBStr(block, adf.OffsetName, adf.MaxFilenameLen, db.Name)

	// Hash chain at offset -4
	adf.PutLong(block, -4, db.HashChain)

	// Parent at offset -3
	adf.PutLong(block, -3, db.Parent)

	// Extension at offset -2: 0 for OFS
	adf.PutLong(block, -2, 0)

	// Secondary type at offset -1: ST_USERDIR (2)
	adf.PutLong(block, -1, adf.STUserDir)

	// Calculate and set checksum at offset 5
	adf.SetBlockChecksum(block, 5)

	return block
}

// WriteBlock writes the directory block to disk
func (db *DirBlock) WriteBlock(disk *adf.Disk) error {
	block := db.CreateBlock()
	return disk.WriteBlock(int(db.BlockNum), block)
}

