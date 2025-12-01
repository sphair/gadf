package ofs

import (
	"testing"

	"github.com/sphair/gadf/pkg/adf"
)

func TestBitmapAllocate(t *testing.T) {
	bm := NewBitmap()

	if bm.IsAllocated(2) {
		t.Error("Block 2 should be free initially")
	}

	bm.Allocate(2)
	if !bm.IsAllocated(2) {
		t.Error("Block 2 should be allocated after Allocate()")
	}

	bm.Free(2)
	if bm.IsAllocated(2) {
		t.Error("Block 2 should be free after Free()")
	}
}

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

	if !bm.IsAllocated(first) {
		t.Error("AllocateNext() block should be marked allocated")
	}

	second := bm.AllocateNext()
	if second == first {
		t.Error("Second allocation should be different from first")
	}
}

func TestBitmapAllocateSpecific(t *testing.T) {
	bm := NewBitmap()

	if !bm.AllocateSpecific(500) {
		t.Error("AllocateSpecific(500) should succeed on free block")
	}

	if !bm.IsAllocated(500) {
		t.Error("Block 500 should be allocated")
	}

	if bm.AllocateSpecific(500) {
		t.Error("AllocateSpecific(500) should fail on allocated block")
	}
}

func TestBitmapFull(t *testing.T) {
	bm := NewBitmap()

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

	if bm.AllocateNext() != -1 {
		t.Error("AllocateNext() should return -1 when disk is full")
	}
}

// TestBitmapDiskFormat verifies on-disk format: bit N = block N, set=free, clear=allocated
func TestBitmapDiskFormat(t *testing.T) {
	bm := NewBitmap()

	bm.AllocateSpecific(adf.RootBlock)
	bm.AllocateSpecific(100)

	block := bm.CreateBitmapBlock()

	isBitClear := func(blockNum int) bool {
		longIdx := blockNum / 32
		bitPos := uint(blockNum % 32)
		longVal := adf.GetLong(block, adf.BitmapDataStartLong+longIdx)
		return (longVal & (1 << bitPos)) == 0
	}

	if !isBitClear(0) || !isBitClear(1) {
		t.Error("Bootblock (0-1) should be allocated")
	}
	if !isBitClear(100) {
		t.Error("Block 100 should be allocated")
	}
	if !isBitClear(adf.RootBlock) {
		t.Errorf("Block %d (rootblock) should be allocated", adf.RootBlock)
	}
	if isBitClear(2) {
		t.Error("Block 2 should be free")
	}
	if isBitClear(500) {
		t.Error("Block 500 should be free")
	}
}
