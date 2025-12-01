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
	adf.PutLong(block, adf.OffsetBlockType, adf.TypeHeader)

	// Header key at offset 4: unused for rootblock (0)
	adf.PutLong(block, adf.OffsetHeaderKey, 0)

	// High seq at offset 8: unused (0)
	adf.PutLong(block, adf.OffsetHighSeq, 0)

	// Hash table size at offset 12: 72
	adf.PutLong(block, adf.OffsetHashTableSize, HashTableSize)

	// First data at offset 16: unused (0)
	adf.PutLong(block, adf.OffsetFirstDataLong, 0)

	// Checksum at offset 20: calculated later
	// adf.PutLong(block, 5, 0)

	// Hash table at offset 24 (72 entries)
	for i := 0; i < HashTableSize; i++ {
		adf.PutLong(block, adf.OffsetHashTable+i, rb.HashTable[i])
	}

	// Bitmap flag
	adf.PutLong(block, adf.OffsetBitmapFlag, rb.BitmapFlag)

	// Bitmap pages (first entry used on DD floppy)
	adf.PutLong(block, adf.OffsetBitmapPages, rb.BitmapBlock)
	// Rest are 0 for floppy

	// Bitmap extension block (unused for floppy)
	adf.PutLong(block, adf.OffsetBitmapExtension, 0)

	// Root alteration date
	days, mins, ticks := timeToAmiga(rb.ModifyTime)
	adf.PutLong(block, adf.OffsetDateDays, days)
	adf.PutLong(block, adf.OffsetDateMins, mins)
	adf.PutLong(block, adf.OffsetDateTicks, ticks)

	// Volume name
	adf.PutBStr(block, adf.OffsetName, adf.MaxFilenameLen, rb.Name)

	// Disk alteration date
	days, mins, ticks = timeToAmiga(rb.DiskTime)
	adf.PutLong(block, adf.OffsetDiskDateDays, days)
	adf.PutLong(block, adf.OffsetDiskDateMins, mins)
	adf.PutLong(block, adf.OffsetDiskDateTicks, ticks)

	// Creation date
	days, mins, ticks = timeToAmiga(rb.CreateTime)
	adf.PutLong(block, adf.OffsetCreateDateDays, days)
	adf.PutLong(block, adf.OffsetCreateDateMins, mins)
	adf.PutLong(block, adf.OffsetCreateDateTicks, ticks)

	// Next hash: unused (0)
	adf.PutLong(block, adf.OffsetNextHash, 0)

	// Parent: unused (0)
	adf.PutLong(block, adf.OffsetParent, 0)

	// Extension (dircache) not used for OFS
	adf.PutLong(block, adf.OffsetExtension, 0)

	// Secondary type: ST_ROOT
	adf.PutLong(block, adf.OffsetSecondaryType, adf.STRoot)

	// Calculate and set checksum at offset 5
	adf.SetBlockChecksum(block, adf.OffsetChecksum)

	return block
}

// WriteBlock writes the root block to disk
func (rb *RootBlock) WriteBlock(disk *adf.Disk) error {
	block := rb.CreateBlock()
	return disk.WriteBlock(adf.RootBlock, block)
}
