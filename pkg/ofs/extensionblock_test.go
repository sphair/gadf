package ofs

import (
	"testing"

	"github.com/sphair/gadf/pkg/adf"
)

func TestNewExtensionBlock(t *testing.T) {
	eb := NewExtensionBlock(100, 50)

	if eb.BlockNum != 100 {
		t.Errorf("BlockNum = %d, want 100", eb.BlockNum)
	}
	if eb.HeaderKey != 50 {
		t.Errorf("HeaderKey = %d, want 50", eb.HeaderKey)
	}
	if eb.Extension != 0 {
		t.Errorf("Extension = %d, want 0", eb.Extension)
	}
	if len(eb.DataBlocks) != 0 {
		t.Errorf("DataBlocks length = %d, want 0", len(eb.DataBlocks))
	}
	if cap(eb.DataBlocks) != MaxDataBlocks {
		t.Errorf("DataBlocks capacity = %d, want %d", cap(eb.DataBlocks), MaxDataBlocks)
	}
}

func TestExtensionBlockAddDataBlock(t *testing.T) {
	eb := NewExtensionBlock(100, 50)

	// Add blocks up to max
	for i := 0; i < MaxDataBlocks; i++ {
		ok := eb.AddDataBlock(uint32(200 + i))
		if !ok {
			t.Errorf("AddDataBlock(%d) returned false, want true", i)
		}
	}

	if len(eb.DataBlocks) != MaxDataBlocks {
		t.Errorf("DataBlocks length = %d, want %d", len(eb.DataBlocks), MaxDataBlocks)
	}

	// Next add should fail
	ok := eb.AddDataBlock(999)
	if ok {
		t.Error("AddDataBlock beyond max returned true, want false")
	}

	// Verify the data blocks are stored correctly
	for i := 0; i < MaxDataBlocks; i++ {
		if eb.DataBlocks[i] != uint32(200+i) {
			t.Errorf("DataBlocks[%d] = %d, want %d", i, eb.DataBlocks[i], 200+i)
		}
	}
}

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
	blockType := adf.GetLong(block, 0)
	if blockType != adf.TypeList {
		t.Errorf("Block type = %d, want %d (T_LIST)", blockType, adf.TypeList)
	}

	// Verify header key at offset 4
	headerKey := adf.GetLong(block, 1)
	if headerKey != 50 {
		t.Errorf("Header key = %d, want 50", headerKey)
	}

	// Verify high seq at offset 8 (number of data blocks)
	highSeq := adf.GetLong(block, 2)
	if highSeq != 3 {
		t.Errorf("High seq = %d, want 3", highSeq)
	}

	// Verify data blocks in reverse order at offset -51, -52, -53
	db1 := adf.GetLong(block, -51)
	if db1 != 201 {
		t.Errorf("Data block 1 at -51 = %d, want 201", db1)
	}
	db2 := adf.GetLong(block, -52)
	if db2 != 202 {
		t.Errorf("Data block 2 at -52 = %d, want 202", db2)
	}
	db3 := adf.GetLong(block, -53)
	if db3 != 203 {
		t.Errorf("Data block 3 at -53 = %d, want 203", db3)
	}

	// Verify extension at offset -2
	ext := adf.GetLong(block, -2)
	if ext != 150 {
		t.Errorf("Extension = %d, want 150", ext)
	}

	// Verify secondary type at offset -1: ST_FILE (-3 as uint32)
	secType := adf.GetLong(block, -1)
	if secType != adf.STFile {
		t.Errorf("Secondary type = %d, want %d (ST_FILE)", secType, adf.STFile)
	}

	// Verify checksum is valid
	storedChecksum := adf.GetLong(block, 5)
	calculatedChecksum := adf.CalculateChecksum(block, 5)
	if storedChecksum != calculatedChecksum {
		t.Errorf("Block checksum invalid: stored %d, calculated %d", storedChecksum, calculatedChecksum)
	}
}

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

	ext1 := adf.GetLong(block1, -2)
	if ext1 != 101 {
		t.Errorf("eb1 extension = %d, want 101", ext1)
	}

	ext2 := adf.GetLong(block2, -2)
	if ext2 != 102 {
		t.Errorf("eb2 extension = %d, want 102", ext2)
	}

	ext3 := adf.GetLong(block3, -2)
	if ext3 != 0 {
		t.Errorf("eb3 extension = %d, want 0 (end of chain)", ext3)
	}

	// All should point to same header
	for i, block := range [][]byte{block1, block2, block3} {
		headerKey := adf.GetLong(block, 1)
		if headerKey != 50 {
			t.Errorf("block%d header key = %d, want 50", i+1, headerKey)
		}
	}
}

func TestExtensionBlockMaxDataBlocks(t *testing.T) {
	eb := NewExtensionBlock(100, 50)

	// Fill to capacity
	for i := 0; i < MaxDataBlocks; i++ {
		eb.AddDataBlock(uint32(200 + i))
	}

	block := eb.CreateBlock()

	// Verify high seq equals MaxDataBlocks
	highSeq := adf.GetLong(block, 2)
	if highSeq != MaxDataBlocks {
		t.Errorf("High seq = %d, want %d", highSeq, MaxDataBlocks)
	}

	// Verify all data blocks are in correct positions (reverse order)
	for i := 0; i < MaxDataBlocks; i++ {
		db := adf.GetLong(block, -51-i)
		expected := uint32(200 + i)
		if db != expected {
			t.Errorf("Data block at -%d = %d, want %d", 51+i, db, expected)
		}
	}

	// Verify checksum is still valid with full block
	storedChecksum := adf.GetLong(block, 5)
	calculatedChecksum := adf.CalculateChecksum(block, 5)
	if storedChecksum != calculatedChecksum {
		t.Errorf("Block checksum invalid: stored %d, calculated %d", storedChecksum, calculatedChecksum)
	}
}

func TestExtensionBlockEmptyDataBlocks(t *testing.T) {
	eb := NewExtensionBlock(100, 50)
	// Don't add any data blocks

	block := eb.CreateBlock()

	// High seq should be 0
	highSeq := adf.GetLong(block, 2)
	if highSeq != 0 {
		t.Errorf("High seq = %d, want 0", highSeq)
	}

	// Type and secondary type should still be correct
	blockType := adf.GetLong(block, 0)
	if blockType != adf.TypeList {
		t.Errorf("Block type = %d, want %d", blockType, adf.TypeList)
	}

	secType := adf.GetLong(block, -1)
	if secType != adf.STFile {
		t.Errorf("Secondary type = %d, want %d", secType, adf.STFile)
	}

	storedChecksum := adf.GetLong(block, 5)
	calculatedChecksum := adf.CalculateChecksum(block, 5)
	if storedChecksum != calculatedChecksum {
		t.Errorf("Block checksum invalid: stored %d, calculated %d", storedChecksum, calculatedChecksum)
	}
}

