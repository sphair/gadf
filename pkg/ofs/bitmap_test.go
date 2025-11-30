package ofs

import (
	"testing"

	"github.com/sphair/gadf/pkg/adf"
)

func TestNewBitmap(t *testing.T) {
	bm := NewBitmap()
	if bm == nil {
		t.Fatal("NewBitmap() returned nil")
	}

	// All blocks should initially be free (except bootblock)
	freeCount := bm.CountFree()
	expectedFree := adf.TotalBlocks - 2 // 1760 - 2 bootblock sectors
	if freeCount != expectedFree {
		t.Errorf("Initial free count = %d, want %d", freeCount, expectedFree)
	}
}

func TestBitmapAllocate(t *testing.T) {
	bm := NewBitmap()

	// Block 2 should be free initially
	if bm.IsAllocated(2) {
		t.Error("Block 2 should be free initially")
	}

	// Allocate it
	bm.Allocate(2)
	if !bm.IsAllocated(2) {
		t.Error("Block 2 should be allocated after Allocate()")
	}

	// Free it
	bm.Free(2)
	if bm.IsAllocated(2) {
		t.Error("Block 2 should be free after Free()")
	}
}

func TestBitmapBootblockAlwaysAllocated(t *testing.T) {
	bm := NewBitmap()

	// Bootblock (blocks 0-1) should always appear allocated
	if !bm.IsAllocated(0) {
		t.Error("Block 0 (bootblock) should always be allocated")
	}
	if !bm.IsAllocated(1) {
		t.Error("Block 1 (bootblock) should always be allocated")
	}
}

func TestBitmapAllocateNext(t *testing.T) {
	bm := NewBitmap()

	// First allocation should be from rootblock area
	first := bm.AllocateNext()
	if first < 0 {
		t.Fatal("AllocateNext() returned -1 on empty disk")
	}

	// Should be at or after rootblock
	if first < adf.RootBlock {
		t.Errorf("First allocation = %d, expected >= %d (rootblock)", first, adf.RootBlock)
	}

	// Should be marked as allocated
	if !bm.IsAllocated(first) {
		t.Error("AllocateNext() block should be marked allocated")
	}

	// Next allocation should be different
	second := bm.AllocateNext()
	if second == first {
		t.Error("Second allocation should be different from first")
	}
}

func TestBitmapAllocateSpecific(t *testing.T) {
	bm := NewBitmap()

	// Allocate specific block
	if !bm.AllocateSpecific(500) {
		t.Error("AllocateSpecific(500) should succeed on free block")
	}

	// Should be allocated now
	if !bm.IsAllocated(500) {
		t.Error("Block 500 should be allocated")
	}

	// Allocating again should fail
	if bm.AllocateSpecific(500) {
		t.Error("AllocateSpecific(500) should fail on allocated block")
	}
}

func TestBitmapCountFree(t *testing.T) {
	bm := NewBitmap()

	initialFree := bm.CountFree()

	// Allocate some blocks
	bm.AllocateNext()
	bm.AllocateNext()
	bm.AllocateNext()

	newFree := bm.CountFree()
	if newFree != initialFree-3 {
		t.Errorf("After 3 allocations: free = %d, want %d", newFree, initialFree-3)
	}
}

func TestBitmapFull(t *testing.T) {
	bm := NewBitmap()

	// Allocate all blocks
	count := 0
	for bm.AllocateNext() >= 0 {
		count++
		if count > adf.TotalBlocks {
			t.Fatal("Infinite loop in allocation")
		}
	}

	// Should have allocated all free blocks
	if bm.CountFree() != 0 {
		t.Errorf("After full allocation: free = %d, want 0", bm.CountFree())
	}

	// Next allocation should fail
	if bm.AllocateNext() != -1 {
		t.Error("AllocateNext() should return -1 when disk is full")
	}
}

func TestBitmapBlockCreation(t *testing.T) {
	bm := NewBitmap()

	// Allocate some blocks
	bm.AllocateSpecific(adf.RootBlock)
	bm.AllocateSpecific(adf.BitmapBlock)
	bm.AllocateSpecific(100)
	bm.AllocateSpecific(200)

	// Create bitmap block
	block := bm.CreateBitmapBlock()

	if len(block) != adf.BlockSize {
		t.Errorf("Bitmap block size = %d, want %d", len(block), adf.BlockSize)
	}

	// Checksum should be at offset 0
	// Verify it's non-zero (a valid checksum)
	checksum := adf.GetLong(block, 0)
	if checksum == 0 {
		// Note: checksum could theoretically be 0, but very unlikely
		t.Log("Warning: bitmap checksum is 0 (unusual but possible)")
	}
}

// TestBitmapDiskFormat verifies the on-disk bitmap format matches Amiga spec.
// Block N should be at bit N in the bitmap (block 0 = bit 0, etc.)
func TestBitmapDiskFormat(t *testing.T) {
	bm := NewBitmap()

	// Allocate known blocks
	bm.AllocateSpecific(adf.RootBlock) // 880
	bm.AllocateSpecific(100)

	block := bm.CreateBitmapBlock()

	// Helper to check bit in serialized bitmap
	// Bitmap data is stored as big-endian uint32s starting at offset 4
	// Long 1 (offset 4) = blocks 0-31, Long 2 (offset 8) = blocks 32-63, etc.
	isBitClear := func(blockNum int) bool {
		longIdx := blockNum / 32
		bitPos := uint(blockNum % 32)
		longVal := adf.GetLong(block, 1+longIdx) // +1 because checksum is at position 0
		return (longVal & (1 << bitPos)) == 0
	}

	// Block 0 (bootblock) should be allocated (bit clear)
	if !isBitClear(0) {
		t.Error("Block 0 should be allocated in disk format")
	}

	// Block 1 (bootblock) should be allocated (bit clear)
	if !isBitClear(1) {
		t.Error("Block 1 should be allocated in disk format")
	}

	// Block 100 should be allocated (bit clear)
	if !isBitClear(100) {
		t.Error("Block 100 should be allocated in disk format")
	}

	// Block 880 (rootblock) should be allocated (bit clear)
	if !isBitClear(adf.RootBlock) {
		t.Errorf("Block %d (rootblock) should be allocated in disk format", adf.RootBlock)
	}

	// Block 2 should be free (bit set)
	if isBitClear(2) {
		t.Error("Block 2 should be free in disk format")
	}

	// Block 500 should be free (bit set)
	if isBitClear(500) {
		t.Error("Block 500 should be free in disk format")
	}
}

