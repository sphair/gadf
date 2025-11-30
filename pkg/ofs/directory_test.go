package ofs

import (
	"testing"
	"time"

	"github.com/sphair/gadf/pkg/adf"
)

// TestDirBlockCreateBlock verifies directory block layout matches Amiga OFS spec
// Note: Directory blocks differ from root blocks (have self-pointer, parent, hash chain)
func TestDirBlockCreateBlock(t *testing.T) {
	db := NewDirBlock(100, 880, "TestDir")
	db.HashTable[0] = 150
	db.HashTable[71] = 250
	db.HashChain = 99
	db.Protection = 0x0F

	block := db.CreateBlock()

	// Verify block size
	if len(block) != adf.BlockSize {
		t.Fatalf("Block size = %d, want %d", len(block), adf.BlockSize)
	}

	// Verify type at offset 0: T_HEADER (2)
	if got := adf.GetLong(block, 0); got != adf.TypeHeader {
		t.Errorf("Type = %d, want %d (T_HEADER)", got, adf.TypeHeader)
	}

	// Verify header key (self pointer) at offset 1 - differs from rootblock!
	if got := adf.GetLong(block, 1); got != 100 {
		t.Errorf("Header key = %d, want 100", got)
	}

	// Verify hash table entries (offset 6-77)
	if got := adf.GetLong(block, 6); got != 150 {
		t.Errorf("HashTable[0] = %d, want 150", got)
	}
	if got := adf.GetLong(block, 6+71); got != 250 {
		t.Errorf("HashTable[71] = %d, want 250", got)
	}

	// Verify protection at offset -48
	if got := adf.GetLong(block, -48); got != 0x0F {
		t.Errorf("Protection = 0x%X, want 0x0F", got)
	}

	// Verify directory name
	name := adf.GetBStr(block, adf.OffsetName, adf.MaxFilenameLen)
	if name != "TestDir" {
		t.Errorf("Name = %s, want TestDir", name)
	}

	// Verify hash chain at offset -4
	if got := adf.GetLong(block, -4); got != 99 {
		t.Errorf("Hash chain = %d, want 99", got)
	}

	// Verify parent at offset -3
	if got := adf.GetLong(block, -3); got != 880 {
		t.Errorf("Parent = %d, want 880", got)
	}

	// Verify extension at offset -2: 0 for OFS
	if got := adf.GetLong(block, -2); got != 0 {
		t.Errorf("Extension = %d, want 0", got)
	}

	// Verify secondary type at offset -1: ST_USERDIR (2) - differs from rootblock!
	if got := adf.GetLong(block, -1); got != adf.STUserDir {
		t.Errorf("Secondary type = %d, want %d (ST_USERDIR)", got, adf.STUserDir)
	}

	// Verify checksum is valid
	checksum := adf.GetLong(block, 5)
	calculated := adf.CalculateChecksum(block, 5)
	if checksum != calculated {
		t.Errorf("Checksum mismatch: stored 0x%X, calculated 0x%X", checksum, calculated)
	}
}

// TestDirBlockDateFields verifies directory has only 1 date field (unlike root's 3)
func TestDirBlockDateFields(t *testing.T) {
	fixedTime := time.Date(1990, 3, 20, 10, 15, 30, 0, time.UTC)
	db := NewDirBlock(100, 880, "Test")
	db.ModifyTime = fixedTime

	block := db.CreateBlock()

	days, mins, ticks := timeToAmiga(fixedTime)

	// Check modification date at offset -23, -22, -21
	if got := adf.GetLong(block, -23); got != days {
		t.Errorf("ModifyTime days = %d, want %d", got, days)
	}
	if got := adf.GetLong(block, -22); got != mins {
		t.Errorf("ModifyTime mins = %d, want %d", got, mins)
	}
	if got := adf.GetLong(block, -21); got != ticks {
		t.Errorf("ModifyTime ticks = %d, want %d", got, ticks)
	}
}
