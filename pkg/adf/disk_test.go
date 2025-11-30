package adf

import (
	"os"
	"path/filepath"
	"testing"
)

// TestReadWriteBlock verifies block I/O correctness
func TestReadWriteBlock(t *testing.T) {
	disk := NewDisk()

	testData := make([]byte, BlockSize)
	for i := range testData {
		testData[i] = byte(i)
	}

	if err := disk.WriteBlock(100, testData); err != nil {
		t.Fatalf("WriteBlock failed: %v", err)
	}

	readData, err := disk.ReadBlock(100)
	if err != nil {
		t.Fatalf("ReadBlock failed: %v", err)
	}

	for i := range testData {
		if readData[i] != testData[i] {
			t.Errorf("Data mismatch at byte %d: got %d, want %d", i, readData[i], testData[i])
		}
	}
}

// TestBlockBoundaries verifies boundary checking
func TestBlockBoundaries(t *testing.T) {
	disk := NewDisk()
	testData := make([]byte, BlockSize)

	// Invalid block numbers
	if err := disk.WriteBlock(-1, testData); err == nil {
		t.Error("WriteBlock(-1) should fail")
	}
	if err := disk.WriteBlock(TotalBlocks, testData); err == nil {
		t.Error("WriteBlock(TotalBlocks) should fail")
	}
	if _, err := disk.ReadBlock(-1); err == nil {
		t.Error("ReadBlock(-1) should fail")
	}
	if _, err := disk.ReadBlock(TotalBlocks); err == nil {
		t.Error("ReadBlock(TotalBlocks) should fail")
	}

	// Valid boundaries
	if err := disk.WriteBlock(0, testData); err != nil {
		t.Errorf("WriteBlock(0) failed: %v", err)
	}
	if err := disk.WriteBlock(TotalBlocks-1, testData); err != nil {
		t.Errorf("WriteBlock(TotalBlocks-1) failed: %v", err)
	}
}

// TestWriteBlockSize verifies block size validation
func TestWriteBlockSize(t *testing.T) {
	disk := NewDisk()

	smallData := make([]byte, BlockSize-1)
	if err := disk.WriteBlock(0, smallData); err == nil {
		t.Error("WriteBlock with small data should fail")
	}

	largeData := make([]byte, BlockSize+1)
	if err := disk.WriteBlock(0, largeData); err == nil {
		t.Error("WriteBlock with large data should fail")
	}
}

// TestSaveLoad verifies disk image persistence
func TestSaveLoad(t *testing.T) {
	disk := NewDisk()

	testData := make([]byte, BlockSize)
	for i := range testData {
		testData[i] = byte(i ^ 0xAA)
	}
	disk.WriteBlock(RootBlock, testData)

	tmpFile := filepath.Join(t.TempDir(), "test.adf")
	if err := disk.Save(tmpFile); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	// Check file size
	info, err := os.Stat(tmpFile)
	if err != nil {
		t.Fatalf("Stat failed: %v", err)
	}
	if info.Size() != DiskSize {
		t.Errorf("Saved file size = %d, want %d", info.Size(), DiskSize)
	}

	// Load into new disk
	disk2 := NewDisk()
	if err := disk2.Load(tmpFile); err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	// Verify data
	readData, _ := disk2.ReadBlock(RootBlock)
	for i := range testData {
		if readData[i] != testData[i] {
			t.Errorf("Loaded data mismatch at byte %d: got %d, want %d", i, readData[i], testData[i])
		}
	}
}

// TestWriteBootBlock verifies bootblock write spans 2 blocks
func TestWriteBootBlock(t *testing.T) {
	disk := NewDisk()

	bootData := make([]byte, BlockSize*2)
	bootData[0] = 'D'
	bootData[1] = 'O'
	bootData[2] = 'S'
	bootData[3] = 0

	if err := disk.WriteBootBlock(bootData); err != nil {
		t.Fatalf("WriteBootBlock failed: %v", err)
	}

	if disk.Data[0] != 'D' || disk.Data[1] != 'O' || disk.Data[2] != 'S' {
		t.Error("Bootblock not written at correct position")
	}
}
