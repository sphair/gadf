package ofs

import (
	"testing"
)

func TestSplitFileData(t *testing.T) {
	tests := []struct {
		name           string
		dataSize       int
		expectedChunks int
		lastChunkSize  int
	}{
		{"empty", 0, 1, 0},
		{"small", 100, 1, 100},
		{"exact one block", OFSDataSize, 1, OFSDataSize},
		{"one byte over", OFSDataSize + 1, 2, 1},
		{"two blocks", OFSDataSize * 2, 2, OFSDataSize},
		{"two and a half", OFSDataSize*2 + 100, 3, 100},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data := make([]byte, tt.dataSize)
			for i := range data {
				data[i] = byte(i % 256)
			}

			chunks := SplitFileData(data)

			if len(chunks) != tt.expectedChunks {
				t.Errorf("SplitFileData(%d bytes) = %d chunks, want %d",
					tt.dataSize, len(chunks), tt.expectedChunks)
			}

			if len(chunks) > 0 {
				lastSize := len(chunks[len(chunks)-1])
				if lastSize != tt.lastChunkSize {
					t.Errorf("Last chunk size = %d, want %d", lastSize, tt.lastChunkSize)
				}
			}

			// Verify no chunk exceeds max size
			for i, chunk := range chunks {
				if len(chunk) > OFSDataSize {
					t.Errorf("Chunk %d size = %d, exceeds max %d", i, len(chunk), OFSDataSize)
				}
			}

			// Verify total data is preserved
			totalSize := 0
			for _, chunk := range chunks {
				totalSize += len(chunk)
			}
			if totalSize != tt.dataSize {
				t.Errorf("Total chunk size = %d, want %d", totalSize, tt.dataSize)
			}
		})
	}
}

func TestSplitFileDataContent(t *testing.T) {
	// Create data with recognizable pattern
	data := make([]byte, OFSDataSize+100)
	for i := range data {
		data[i] = byte(i % 256)
	}

	chunks := SplitFileData(data)

	// Verify content is preserved
	offset := 0
	for _, chunk := range chunks {
		for i, b := range chunk {
			expected := byte((offset + i) % 256)
			if b != expected {
				t.Errorf("Data mismatch at offset %d: got %d, want %d", offset+i, b, expected)
			}
		}
		offset += len(chunk)
	}
}

func TestNewDataBlock(t *testing.T) {
	data := []byte{1, 2, 3, 4, 5}
	db := NewDataBlock(100, 50, 1, data)

	if db.BlockNum != 100 {
		t.Errorf("BlockNum = %d, want 100", db.BlockNum)
	}
	if db.HeaderKey != 50 {
		t.Errorf("HeaderKey = %d, want 50", db.HeaderKey)
	}
	if db.SeqNum != 1 {
		t.Errorf("SeqNum = %d, want 1", db.SeqNum)
	}
	if db.DataSize != 5 {
		t.Errorf("DataSize = %d, want 5", db.DataSize)
	}
	if db.NextData != 0 {
		t.Errorf("NextData = %d, want 0", db.NextData)
	}

	// Verify data is copied, not referenced
	data[0] = 99
	if db.Data[0] == 99 {
		t.Error("Data should be copied, not referenced")
	}
}

func TestDataBlockCreate(t *testing.T) {
	data := []byte("Hello, Amiga!")
	db := NewDataBlock(100, 50, 1, data)
	db.NextData = 101

	block := db.CreateBlock()

	if len(block) != 512 {
		t.Errorf("Block size = %d, want 512", len(block))
	}

	// Verify header fields (big-endian)
	// Type at offset 0
	if block[0] != 0 || block[1] != 0 || block[2] != 0 || block[3] != 8 {
		t.Error("Block type should be T_DATA (8)")
	}

	// Data at offset 24
	for i, b := range data {
		if block[24+i] != b {
			t.Errorf("Data mismatch at offset %d: got %d, want %d", 24+i, block[24+i], b)
		}
	}
}

