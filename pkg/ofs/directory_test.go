package ofs

import (
	"testing"
	"time"

	"github.com/sphair/gadf/pkg/adf"
)

// TestDirBlockCreateBlock verifies directory block layout (differs from root: has self-pointer, parent, hash chain)
func TestDirBlockCreateBlock(t *testing.T) {
	db := NewDirBlock(100, 880, "TestDir")
	db.HashTable[0] = 150
	db.HashTable[71] = 250
	db.HashChain = 99
	db.Protection = 0x0F

	block := db.CreateBlock()

	if len(block) != adf.BlockSize {
		t.Fatalf("Block size = %d, want %d", len(block), adf.BlockSize)
	}

	if got := adf.GetLong(block, adf.OffsetBlockType); got != adf.TypeHeader {
		t.Errorf("Type = %d, want %d (T_HEADER)", got, adf.TypeHeader)
	}
	if got := adf.GetLong(block, adf.OffsetHeaderKey); got != 100 {
		t.Errorf("Header key = %d, want 100", got)
	}
	if got := adf.GetLong(block, adf.OffsetHashTable); got != 150 {
		t.Errorf("HashTable[0] = %d, want 150", got)
	}
	if got := adf.GetLong(block, adf.OffsetHashTable+71); got != 250 {
		t.Errorf("HashTable[71] = %d, want 250", got)
	}
	if got := adf.GetLong(block, adf.OffsetProtection); got != 0x0F {
		t.Errorf("Protection = 0x%X, want 0x0F", got)
	}

	nameOffset := adf.BlockSize + adf.OffsetName*4
	if got := string(block[nameOffset : nameOffset+8]); got != "\x07TestDir" {
		t.Errorf("BCPL name = %q, want %q", got, "\x07TestDir")
	}

	if got := adf.GetLong(block, adf.OffsetNextHash); got != 99 {
		t.Errorf("Hash chain = %d, want 99", got)
	}
	if got := adf.GetLong(block, adf.OffsetParent); got != 880 {
		t.Errorf("Parent = %d, want 880", got)
	}
	if got := adf.GetLong(block, adf.OffsetExtension); got != 0 {
		t.Errorf("Extension = %d, want 0", got)
	}
	if got := adf.GetLong(block, adf.OffsetSecondaryType); got != adf.STUserDir {
		t.Errorf("Secondary type = %d, want %d (ST_USERDIR)", got, adf.STUserDir)
	}

	checksum := adf.GetLong(block, adf.OffsetChecksum)
	calculated := adf.CalculateChecksum(block, adf.OffsetChecksum)
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

	if got := adf.GetLong(block, adf.OffsetDateDays); got != days {
		t.Errorf("ModifyTime days = %d, want %d", got, days)
	}
	if got := adf.GetLong(block, adf.OffsetDateMins); got != mins {
		t.Errorf("ModifyTime mins = %d, want %d", got, mins)
	}
	if got := adf.GetLong(block, adf.OffsetDateTicks); got != ticks {
		t.Errorf("ModifyTime ticks = %d, want %d", got, ticks)
	}
}
