package ofs

import (
	"github.com/sphair/gadf/pkg/adf"
)

// Bitmap manages block allocation for the disk
// Bit set = free, bit clear = allocated
type Bitmap struct {
	blocks []uint32 // Bitmap data (each uint32 tracks 32 blocks)
}

// NewBitmap creates a new bitmap for a floppy disk
func NewBitmap() *Bitmap {
	// DD floppy: 1760 blocks = 55 uint32s (55 * 32 = 1760 bits)
	bm := &Bitmap{
		blocks: make([]uint32, 55),
	}
	// All bits set = all blocks free
	for i := range bm.blocks {
		bm.blocks[i] = adf.BitmapValid
	}
	// Pre-allocate bootblock (blocks 0-1)
	bm.Allocate(0)
	bm.Allocate(1)
	return bm
}

// blockToBitmapPos converts a block number to bitmap position
func blockToBitmapPos(blockNum int) (int, uint) {
	return blockNum / 32, uint(blockNum % 32)
}

// IsAllocated checks if a block is allocated
func (bm *Bitmap) IsAllocated(blockNum int) bool {
	if blockNum < 0 || blockNum >= adf.TotalBlocks {
		return true // Invalid blocks treated as allocated
	}
	idx, bit := blockToBitmapPos(blockNum)
	return (bm.blocks[idx] & (1 << bit)) == 0
}

// Allocate marks a block as allocated
func (bm *Bitmap) Allocate(blockNum int) {
	if blockNum < 0 || blockNum >= adf.TotalBlocks {
		return
	}
	idx, bit := blockToBitmapPos(blockNum)
	bm.blocks[idx] &^= (1 << bit)
}

// Free marks a block as free
func (bm *Bitmap) Free(blockNum int) {
	if blockNum < 0 || blockNum >= adf.TotalBlocks {
		return
	}
	idx, bit := blockToBitmapPos(blockNum)
	bm.blocks[idx] |= (1 << bit)
}

// AllocateNext finds and allocates the next free block
// Returns -1 if no free blocks available
// AllocateNext finds next free block via linear search.
// Simple but not optimal - could use a free list for large disks.
func (bm *Bitmap) AllocateNext() int {
	// Start search from rootblock area
	for blockNum := adf.RootBlock; blockNum < adf.TotalBlocks; blockNum++ {
		if !bm.IsAllocated(blockNum) {
			bm.Allocate(blockNum)
			return blockNum
		}
	}
	// Wrap around to beginning (after bootblock)
	for blockNum := 2; blockNum < adf.RootBlock; blockNum++ {
		if !bm.IsAllocated(blockNum) {
			bm.Allocate(blockNum)
			return blockNum
		}
	}
	return -1 // No free blocks
}

// AllocateSpecific allocates a specific block
// Returns false if already allocated
func (bm *Bitmap) AllocateSpecific(blockNum int) bool {
	if bm.IsAllocated(blockNum) {
		return false
	}
	bm.Allocate(blockNum)
	return true
}

// CountFree returns the number of free blocks
func (bm *Bitmap) CountFree() int {
	count := 0
	for blockNum := 0; blockNum < adf.TotalBlocks; blockNum++ {
		if !bm.IsAllocated(blockNum) {
			count++
		}
	}
	return count
}

// CreateBitmapBlock creates the bitmap block data
func (bm *Bitmap) CreateBitmapBlock() []byte {
	block := make([]byte, adf.BlockSize)

	for i, val := range bm.blocks {
		adf.PutLong(block, adf.BitmapDataStartLong+i, val)
	}

	adf.SetBlockChecksum(block, adf.BitmapChecksumLong)

	return block
}

// WriteBitmapBlock writes the bitmap to disk at the specified block
func (bm *Bitmap) WriteBitmapBlock(disk *adf.Disk, blockNum int) error {
	block := bm.CreateBitmapBlock()
	return disk.WriteBlock(blockNum, block)
}
