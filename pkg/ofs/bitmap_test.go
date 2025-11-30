package ofs

import (
	"testing"

	"github.com/sphair/gadf/pkg/adf"
)

// TestBitmapAllocate verifies basic allocation and free operations
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

// TestBitmapBootblockAlwaysAllocated verifies bootblock invariant
func TestBitmapBootblockAlwaysAllocated(t *testing.T) {
	bm := NewBitmap()

	if !bm.IsAllocated(0) {
		t.Error("Block 0 (bootblock) should always be allocated")
	}
	if !bm.IsAllocated(1) {
		t.Error("Block 1 (bootblock) should always be allocated")
	}
}

// TestBitmapAllocateNext verifies allocation starts from rootblock area
func TestBitmapAllocateNext(t *testing.T) {
	bm := NewBitmap()

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

// TestBitmapAllocateSpecific verifies specific block allocation
func TestBitmapAllocateSpecific(t *testing.T) {
	bm := NewBitmap()

	if !bm.AllocateSpecific(500) {
		t.Error("AllocateSpecific(500) should succeed on free block")
	}

	if !bm.IsAllocated(500) {
		t.Error("Block 500 should be allocated")
	}

	// Allocating again should fail
	if bm.AllocateSpecific(500) {
		t.Error("AllocateSpecific(500) should fail on allocated block")
	}
}

// TestBitmapFull verifies disk-full handling
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

	if bm.CountFree() != 0 {
		t.Errorf("After full allocation: free = %d, want 0", bm.CountFree())
	}

	// Next allocation should fail
	if bm.AllocateNext() != -1 {
		t.Error("AllocateNext() should return -1 when disk is full")
	}
}

// TestBitmapDiskFormat verifies on-disk bitmap format matches Amiga spec
// Block N is at bit N: set=free, clear=allocated
func TestBitmapDiskFormat(t *testing.T) {
	bm := NewBitmap()

	// Allocate known blocks
	bm.AllocateSpecific(adf.RootBlock) // 880
	bm.AllocateSpecific(100)

	block := bm.CreateBitmapBlock()

	// Helper to check bit in serialized bitmap
	// Bitmap data: Long 1 (offset 4) = blocks 0-31, Long 2 = blocks 32-63, etc.
	isBitClear := func(blockNum int) bool {
		longIdx := blockNum / 32
		bitPos := uint(blockNum % 32)
		longVal := adf.GetLong(block, 1+longIdx) // +1 because checksum is at position 0
		return (longVal & (1 << bitPos)) == 0
	}

	// Bootblock should be allocated (bit clear)
	if !isBitClear(0) {
		t.Error("Block 0 should be allocated in disk format")
	}
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
