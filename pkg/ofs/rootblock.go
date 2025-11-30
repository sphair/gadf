package ofs

import (
	"time"

	"github.com/sphair/gadf/pkg/adf"
)

// RootBlock represents the root block of an OFS volume
type RootBlock struct {
	Name        string     // Volume name (max 30 chars)
	HashTable   [72]uint32 // Hash table entries (block pointers)
	BitmapFlag  uint32     // 0xFFFFFFFF = valid bitmap
	BitmapBlock uint32     // Block number of first bitmap block
	CreateTime  time.Time
	ModifyTime  time.Time
	DiskTime    time.Time
}

// NewRootBlock creates a new root block with the given volume name
func NewRootBlock(name string, bitmapBlock uint32) *RootBlock {
	now := time.Now()
	return &RootBlock{
		Name:        name,
		BitmapFlag:  adf.BitmapValid,
		BitmapBlock: bitmapBlock,
		CreateTime:  now,
		ModifyTime:  now,
		DiskTime:    now,
	}
}

// SetHashEntry sets a hash table entry
func (rb *RootBlock) SetHashEntry(index int, blockNum uint32) {
	if index >= 0 && index < HashTableSize {
		rb.HashTable[index] = blockNum
	}
}

// GetHashEntry gets a hash table entry
func (rb *RootBlock) GetHashEntry(index int) uint32 {
	if index >= 0 && index < HashTableSize {
		return rb.HashTable[index]
	}
	return 0
}

// AddHashEntry adds an entry to the hash table and returns the previous entry (for chaining).
// The caller should set the returned value as the new entry's HashChain.
func (rb *RootBlock) AddHashEntry(name string, blockNum uint32) uint32 {
	index := HashName(name)
	previous := rb.HashTable[index]
	rb.HashTable[index] = blockNum
	return previous
}

// timeToAmiga converts a Go time to Amiga timestamp format
// Amiga uses days since Jan 1, 1978, minutes past midnight, and ticks (1/50 sec)
func timeToAmiga(t time.Time) (days, mins, ticks uint32) {
	// Amiga epoch: January 1, 1978
	amigaEpoch := time.Date(1978, 1, 1, 0, 0, 0, 0, time.UTC)

	// Calculate days since Amiga epoch
	duration := t.Sub(amigaEpoch)
	days = uint32(duration.Hours() / 24)

	// Calculate minutes past midnight
	mins = uint32(t.Hour()*60 + t.Minute())

	// Calculate ticks (1/50 second)
	ticks = uint32(t.Second() * 50)

	return
}

// CreateBlock creates the raw block data for the root block
func (rb *RootBlock) CreateBlock() []byte {
	block := make([]byte, adf.BlockSize)

	// Type at offset 0: T_HEADER (2)
	adf.PutLong(block, 0, adf.TypeHeader)

	// Header key at offset 4: unused for rootblock (0)
	adf.PutLong(block, 1, 0)

	// High seq at offset 8: unused (0)
	adf.PutLong(block, 2, 0)

	// Hash table size at offset 12: 72
	adf.PutLong(block, 3, HashTableSize)

	// First data at offset 16: unused (0)
	adf.PutLong(block, 4, 0)

	// Checksum at offset 20: calculated later
	// adf.PutLong(block, 5, 0)

	// Hash table at offset 24 (72 entries)
	for i := 0; i < HashTableSize; i++ {
		adf.PutLong(block, 6+i, rb.HashTable[i])
	}

	// Bitmap flag at offset -50 (BSIZE-200)
	adf.PutLong(block, -50, rb.BitmapFlag)

	// Bitmap pages at offset -49 to -25 (25 entries, first one used)
	adf.PutLong(block, -49, rb.BitmapBlock)
	// Rest are 0 for floppy

	// Bitmap extension block at offset -24: 0 (not used for floppy)
	adf.PutLong(block, -24, 0)

	// Root alteration date at offset -23 to -21
	days, mins, ticks := timeToAmiga(rb.ModifyTime)
	adf.PutLong(block, -23, days)
	adf.PutLong(block, -22, mins)
	adf.PutLong(block, -21, ticks)

	// Volume name
	adf.PutBStr(block, adf.OffsetName, adf.MaxFilenameLen, rb.Name)

	// Disk alteration date at offset -10 to -8
	days, mins, ticks = timeToAmiga(rb.DiskTime)
	adf.PutLong(block, -10, days)
	adf.PutLong(block, -9, mins)
	adf.PutLong(block, -8, ticks)

	// Creation date at offset -7 to -5
	days, mins, ticks = timeToAmiga(rb.CreateTime)
	adf.PutLong(block, -7, days)
	adf.PutLong(block, -6, mins)
	adf.PutLong(block, -5, ticks)

	// Next hash at offset -4: unused (0)
	adf.PutLong(block, -4, 0)

	// Parent at offset -3: unused (0)
	adf.PutLong(block, -3, 0)

	// Extension (dircache) at offset -2: 0 for OFS
	adf.PutLong(block, -2, 0)

	// Secondary type at offset -1: ST_ROOT (1)
	adf.PutLong(block, -1, adf.STRoot)

	// Calculate and set checksum at offset 5
	adf.SetBlockChecksum(block, 5)

	return block
}

// WriteBlock writes the root block to disk
func (rb *RootBlock) WriteBlock(disk *adf.Disk) error {
	block := rb.CreateBlock()
	return disk.WriteBlock(adf.RootBlock, block)
}

