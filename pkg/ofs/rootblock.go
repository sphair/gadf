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

	// Header
	adf.PutLong(block, adf.OffsetBlockType, adf.TypeHeader)
	adf.PutLong(block, adf.OffsetHeaderKey, 0) // unused for root
	adf.PutLong(block, adf.OffsetHighSeq, 0)
	adf.PutLong(block, adf.OffsetHashTableSize, HashTableSize)
	adf.PutLong(block, adf.OffsetFirstDataLong, 0)

	// Hash table (72 entries)
	for i := range HashTableSize {
		adf.PutLong(block, adf.OffsetHashTable+i, rb.HashTable[i])
	}

	// Bitmap info
	adf.PutLong(block, adf.OffsetBitmapFlag, rb.BitmapFlag)
	adf.PutLong(block, adf.OffsetBitmapPages, rb.BitmapBlock)
	adf.PutLong(block, adf.OffsetBitmapExtension, 0)

	// Modification date
	days, mins, ticks := timeToAmiga(rb.ModifyTime)
	adf.PutLong(block, adf.OffsetDateDays, days)
	adf.PutLong(block, adf.OffsetDateMins, mins)
	adf.PutLong(block, adf.OffsetDateTicks, ticks)

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

	// Footer
	adf.PutLong(block, adf.OffsetNextHash, 0)
	adf.PutLong(block, adf.OffsetParent, 0)
	adf.PutLong(block, adf.OffsetExtension, 0)
	adf.PutLong(block, adf.OffsetSecondaryType, adf.STRoot)

	adf.SetBlockChecksum(block, adf.OffsetChecksum)

	return block
}

// WriteBlock writes the root block to disk
func (rb *RootBlock) WriteBlock(disk *adf.Disk) error {
	block := rb.CreateBlock()
	return disk.WriteBlock(adf.RootBlock, block)
}
