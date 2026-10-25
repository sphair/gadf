package ofs

import (
	"testing"

	"github.com/sphair/gadf/pkg/adf"
)

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

	blockType := adf.GetLong(block, adf.OffsetBlockType)
	if blockType != adf.TypeList {
		t.Errorf("Block type = %d, want %d (T_LIST)", blockType, adf.TypeList)
	}

	headerKey := adf.GetLong(block, adf.OffsetHeaderKey)
	if headerKey != 100 {
		t.Errorf("Header key = %d, want 100 (self)", headerKey)
	}

	parent := adf.GetLong(block, adf.OffsetParent)
	if parent != 50 {
		t.Errorf("Parent = %d, want 50 (file header)", parent)
	}

	highSeq := adf.GetLong(block, adf.OffsetHighSeq)
	if highSeq != 3 {
		t.Errorf("High seq = %d, want 3", highSeq)
	}

	db1 := adf.GetLong(block, adf.OffsetFirstDataPtr)
	if db1 != 201 {
		t.Errorf("Data block 1 = %d, want 201", db1)
	}
	db2 := adf.GetLong(block, adf.OffsetFirstDataPtr-1)
	if db2 != 202 {
		t.Errorf("Data block 2 = %d, want 202", db2)
	}
	db3 := adf.GetLong(block, adf.OffsetFirstDataPtr-2)
	if db3 != 203 {
		t.Errorf("Data block 3 = %d, want 203", db3)
	}

	ext := adf.GetLong(block, adf.OffsetExtension)
	if ext != 150 {
		t.Errorf("Extension = %d, want 150", ext)
	}

	secType := adf.GetLong(block, adf.OffsetSecondaryType)
	if secType != adf.STFile {
		t.Errorf("Secondary type = %d, want %d (ST_FILE)", secType, adf.STFile)
	}

	storedChecksum := adf.GetLong(block, adf.OffsetChecksum)
	calculatedChecksum := adf.CalculateChecksum(block, adf.OffsetChecksum)
	if storedChecksum != calculatedChecksum {
		t.Errorf("Block checksum invalid: stored %d, calculated %d", storedChecksum, calculatedChecksum)
	}
}

func TestExtensionBlockChaining(t *testing.T) {
	eb1 := NewExtensionBlock(100, 50)
	eb2 := NewExtensionBlock(101, 50)
	eb3 := NewExtensionBlock(102, 50)

	eb1.Extension = eb2.BlockNum
	eb2.Extension = eb3.BlockNum

	eb1.AddDataBlock(200)
	eb2.AddDataBlock(300)
	eb3.AddDataBlock(400)

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

	for i, block := range [][]byte{block1, block2, block3} {
		if self := adf.GetLong(block, adf.OffsetHeaderKey); self != uint32(100+i) {
			t.Errorf("block%d header key = %d, want %d", i+1, self, 100+i)
		}
		if parent := adf.GetLong(block, adf.OffsetParent); parent != 50 {
			t.Errorf("block%d parent = %d, want 50", i+1, parent)
		}
	}
}

func TestExtensionBlockMaxDataBlocks(t *testing.T) {
	eb := NewExtensionBlock(100, 50)

	for i := range MaxDataBlocks {
		if !eb.AddDataBlock(uint32(200 + i)) {
			t.Fatalf("AddDataBlock failed at %d", i)
		}
	}

	if eb.AddDataBlock(999) {
		t.Error("AddDataBlock beyond max should return false")
	}

	block := eb.CreateBlock()

	highSeq := adf.GetLong(block, adf.OffsetHighSeq)
	if highSeq != MaxDataBlocks {
		t.Errorf("High seq = %d, want %d", highSeq, MaxDataBlocks)
	}

	for i := range MaxDataBlocks {
		db := adf.GetLong(block, adf.OffsetFirstDataPtr-i)
		expected := uint32(200 + i)
		if db != expected {
			t.Errorf("Data block at -%d = %d, want %d", 51+i, db, expected)
		}
	}
}
