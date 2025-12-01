package ofs

import (
	"testing"
)

func TestRequiredExtensionBlocks(t *testing.T) {
	tests := []struct {
		numDataBlocks int
		expected      int
	}{
		{0, 0},
		{1, 0},
		{71, 0},
		{72, 0},    // exactly fits in file header
		{73, 1},    // needs 1 extension block
		{144, 1},   // 72 + 72 = exactly fits header + 1 extension
		{145, 2},   // needs 2 extension blocks
		{216, 2},   // 72 + 72 + 72 = exactly fits
		{217, 3},   // needs 3 extension blocks
		{1000, 13}, // (1000-72+71)/72 = 13
	}

	for _, tt := range tests {
		got := RequiredExtensionBlocks(tt.numDataBlocks)
		if got != tt.expected {
			t.Errorf("RequiredExtensionBlocks(%d) = %d, want %d", tt.numDataBlocks, got, tt.expected)
		}
	}
}

func TestAssignDataBlocksSmallFile(t *testing.T) {
	fh := NewFileHeader(100, 880, "test", 1000)

	dataBlocks := []uint32{200, 201, 202, 203, 204, 205, 206, 207, 208, 209}
	fh.AssignDataBlocks(dataBlocks, nil)

	if len(fh.extensions) != 0 {
		t.Errorf("Expected 0 extension blocks, got %d", len(fh.extensions))
	}

	if len(fh.DataBlocks) != 10 {
		t.Errorf("FileHeader.DataBlocks length = %d, want 10", len(fh.DataBlocks))
	}

	if fh.Extension != 0 {
		t.Errorf("Extension = %d, want 0", fh.Extension)
	}
}

// TestAssignDataBlocksOneExtension verifies overflow goes to extension block
func TestAssignDataBlocksOneExtension(t *testing.T) {
	fh := NewFileHeader(100, 880, "test", 50000)

	// 80 data blocks: 72 in header, 8 in extension
	dataBlocks := make([]uint32, 80)
	for i := range dataBlocks {
		dataBlocks[i] = uint32(200 + i)
	}

	extBlockNums := []uint32{150}
	fh.AssignDataBlocks(dataBlocks, extBlockNums)

	if len(fh.DataBlocks) != MaxDataBlocks {
		t.Errorf("FileHeader.DataBlocks length = %d, want %d", len(fh.DataBlocks), MaxDataBlocks)
	}
	if fh.Extension != 150 {
		t.Errorf("FileHeader.Extension = %d, want 150", fh.Extension)
	}

	if len(fh.extensions) != 1 {
		t.Fatalf("Expected 1 extension block, got %d", len(fh.extensions))
	}

	eb := fh.extensions[0]
	if eb.BlockNum != 150 {
		t.Errorf("ExtensionBlock.BlockNum = %d, want 150", eb.BlockNum)
	}
	if eb.HeaderKey != 100 {
		t.Errorf("ExtensionBlock.HeaderKey = %d, want 100", eb.HeaderKey)
	}
	if len(eb.DataBlocks) != 8 {
		t.Errorf("ExtensionBlock.DataBlocks length = %d, want 8", len(eb.DataBlocks))
	}
	if eb.Extension != 0 {
		t.Errorf("ExtensionBlock.Extension = %d, want 0 (end of chain)", eb.Extension)
	}
}

// TestAssignDataBlocksMultipleExtensions verifies chaining of extension blocks
func TestAssignDataBlocksMultipleExtensions(t *testing.T) {
	fh := NewFileHeader(100, 880, "test", 100000)

	// 200 data blocks: 72 in header, 72 in ext1, 56 in ext2
	dataBlocks := make([]uint32, 200)
	for i := range dataBlocks {
		dataBlocks[i] = uint32(200 + i)
	}

	extBlockNums := []uint32{150, 151}
	fh.AssignDataBlocks(dataBlocks, extBlockNums)

	if fh.Extension != 150 {
		t.Errorf("FileHeader.Extension = %d, want 150", fh.Extension)
	}

	if len(fh.extensions) != 2 {
		t.Fatalf("Expected 2 extension blocks, got %d", len(fh.extensions))
	}

	if fh.extensions[0].Extension != 151 {
		t.Errorf("extensions[0].Extension = %d, want 151", fh.extensions[0].Extension)
	}
	if fh.extensions[1].Extension != 0 {
		t.Errorf("extensions[1].Extension = %d, want 0", fh.extensions[1].Extension)
	}

	if len(fh.DataBlocks) != MaxDataBlocks {
		t.Errorf("FileHeader has %d data blocks, want %d", len(fh.DataBlocks), MaxDataBlocks)
	}
	if len(fh.extensions[0].DataBlocks) != MaxDataBlocks {
		t.Errorf("extensions[0] has %d data blocks, want %d", len(fh.extensions[0].DataBlocks), MaxDataBlocks)
	}
	if len(fh.extensions[1].DataBlocks) != 56 {
		t.Errorf("extensions[1] has %d data blocks, want 56", len(fh.extensions[1].DataBlocks))
	}

	for i, eb := range fh.extensions {
		if eb.HeaderKey != 100 {
			t.Errorf("extensions[%d].HeaderKey = %d, want 100", i, eb.HeaderKey)
		}
	}
}

func TestFileHeaderAddDataBlock(t *testing.T) {
	fh := NewFileHeader(100, 880, "test", 1000)

	for i := 0; i < MaxDataBlocks; i++ {
		ok := fh.AddDataBlock(uint32(200 + i))
		if !ok {
			t.Errorf("AddDataBlock(%d) returned false, want true", i)
		}
	}

	ok := fh.AddDataBlock(999)
	if ok {
		t.Error("AddDataBlock beyond max returned true, want false")
	}

	if len(fh.DataBlocks) != MaxDataBlocks {
		t.Errorf("DataBlocks length = %d, want %d", len(fh.DataBlocks), MaxDataBlocks)
	}
}
