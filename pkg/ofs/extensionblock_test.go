package ofs

import (
	"testing"

	"github.com/sphair/gadf/pkg/adf"
)

// TestExtensionBlockCreateBlock verifies extension block layout
func TestExtensionBlockCreateBlock(t *testing.T) {
	eb := NewExtensionBlock(100, 50)
	eb.AddDataBlock(201)
	eb.AddDataBlock(202)
	eb.AddDataBlock(203)
	eb.Extension = 150

	block := eb.CreateBlock()

	if len(block) != adf.BlockSize {
		t.Errorf("Block size = %d, want %d", len(block), adf.BlockSize)
	}

	// Verify type at offset 0: T_LIST (16)
	blockType := adf.GetLong(block, adf.OffsetBlockType)
	if blockType != adf.TypeList {
		t.Errorf("Block type = %d, want %d (T_LIST)", blockType, adf.TypeList)
	}

	// Verify header key at offset 4
	headerKey := adf.GetLong(block, adf.OffsetHeaderKey)
	if headerKey != 50 {
		t.Errorf("Header key = %d, want 50", headerKey)
	}

	// Verify high seq at offset 8 (number of data blocks)
	highSeq := adf.GetLong(block, adf.OffsetHighSeq)
	if highSeq != 3 {
		t.Errorf("High seq = %d, want 3", highSeq)
	}

	// Verify data blocks in reverse order at defined offsets
	db1 := adf.GetLong(block, adf.OffsetFirstDataPtr)
	if db1 != 201 {
		t.Errorf("Data block 1 at -51 = %d, want 201", db1)
	}
	db2 := adf.GetLong(block, adf.OffsetFirstDataPtr-1)
	if db2 != 202 {
		t.Errorf("Data block 2 at -52 = %d, want 202", db2)
	}
	db3 := adf.GetLong(block, adf.OffsetFirstDataPtr-2)
	if db3 != 203 {
		t.Errorf("Data block 3 at -53 = %d, want 203", db3)
	}

	// Verify extension pointer
	ext := adf.GetLong(block, adf.OffsetExtension)
	if ext != 150 {
		t.Errorf("Extension = %d, want 150", ext)
	}

	// Verify secondary type
	secType := adf.GetLong(block, adf.OffsetSecondaryType)
	if secType != adf.STFile {
		t.Errorf("Secondary type = %d, want %d (ST_FILE)", secType, adf.STFile)
	}

	// Verify checksum is valid
	storedChecksum := adf.GetLong(block, adf.OffsetChecksum)
	calculatedChecksum := adf.CalculateChecksum(block, adf.OffsetChecksum)
	if storedChecksum != calculatedChecksum {
		t.Errorf("Block checksum invalid: stored %d, calculated %d", storedChecksum, calculatedChecksum)
	}
}

// TestExtensionBlockChaining verifies extension blocks can be chained
func TestExtensionBlockChaining(t *testing.T) {
	// Create a chain of extension blocks
	eb1 := NewExtensionBlock(100, 50)
	eb2 := NewExtensionBlock(101, 50)
	eb3 := NewExtensionBlock(102, 50)

	// Chain them
	eb1.Extension = eb2.BlockNum
	eb2.Extension = eb3.BlockNum
	// eb3.Extension stays 0 (end of chain)

	// Add some data blocks to each
	eb1.AddDataBlock(200)
	eb2.AddDataBlock(300)
	eb3.AddDataBlock(400)

	// Verify chain in created blocks
	block1 := eb1.CreateBlock()
	block2 := eb2.CreateBlock()
	block3 := eb3.CreateBlock()

	ext1 := adf.GetLong(block1, adf.OffsetExtension)
	if ext1 != 101 {
		t.Errorf("eb1 extension = %d, want 101", ext1)
	}

	ext2 := adf.GetLong(block2, adf.OffsetExtension)
	if ext2 != 102 {
		t.Errorf("eb2 extension = %d, want 102", ext2)
	}

	ext3 := adf.GetLong(block3, adf.OffsetExtension)
	if ext3 != 0 {
		t.Errorf("eb3 extension = %d, want 0 (end of chain)", ext3)
	}

	// All should point to same header
	for i, block := range [][]byte{block1, block2, block3} {
		headerKey := adf.GetLong(block, adf.OffsetHeaderKey)
		if headerKey != 50 {
			t.Errorf("block%d header key = %d, want 50", i+1, headerKey)
		}
	}
}

// TestExtensionBlockMaxDataBlocks verifies 72 data blocks can be stored
func TestExtensionBlockMaxDataBlocks(t *testing.T) {
	eb := NewExtensionBlock(100, 50)

	// Fill to capacity
	for i := 0; i < MaxDataBlocks; i++ {
		if !eb.AddDataBlock(uint32(200 + i)) {
			t.Fatalf("AddDataBlock failed at %d", i)
		}
	}

	// 73rd should fail
	if eb.AddDataBlock(999) {
		t.Error("AddDataBlock beyond max should return false")
	}

	block := eb.CreateBlock()

	// Verify high seq equals MaxDataBlocks
	highSeq := adf.GetLong(block, adf.OffsetHighSeq)
	if highSeq != MaxDataBlocks {
		t.Errorf("High seq = %d, want %d", highSeq, MaxDataBlocks)
	}

	// Verify all data blocks are in correct positions (reverse order)
	for i := 0; i < MaxDataBlocks; i++ {
		db := adf.GetLong(block, adf.OffsetFirstDataPtr-i)
		expected := uint32(200 + i)
		if db != expected {
			t.Errorf("Data block at -%d = %d, want %d", 51+i, db, expected)
		}
	}
}
