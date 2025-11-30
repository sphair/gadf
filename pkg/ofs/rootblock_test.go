package ofs

import (
	"testing"
	"time"

	"github.com/sphair/gadf/pkg/adf"
)

func TestTimeToAmiga(t *testing.T) {
	tests := []struct {
		name         string
		time         time.Time
		expectedDays uint32
		expectedMins uint32
		expectedTick uint32
	}{
		{
			name:         "Amiga epoch",
			time:         time.Date(1978, 1, 1, 0, 0, 0, 0, time.UTC),
			expectedDays: 0,
			expectedMins: 0,
			expectedTick: 0,
		},
		{
			name:         "One day after epoch",
			time:         time.Date(1978, 1, 2, 0, 0, 0, 0, time.UTC),
			expectedDays: 1,
			expectedMins: 0,
			expectedTick: 0,
		},
		{
			name:         "Noon on epoch",
			time:         time.Date(1978, 1, 1, 12, 30, 0, 0, time.UTC),
			expectedDays: 0,
			expectedMins: 12*60 + 30,
			expectedTick: 0,
		},
		{
			name:         "With seconds",
			time:         time.Date(1978, 1, 1, 0, 0, 30, 0, time.UTC),
			expectedDays: 0,
			expectedMins: 0,
			expectedTick: 30 * 50, // 50 ticks per second
		},
		{
			name:         "Year 2000",
			time:         time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC),
			expectedDays: 8035, // Days from 1978-01-01 to 2000-01-01
			expectedMins: 0,
			expectedTick: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			days, mins, ticks := timeToAmiga(tt.time)

			if days != tt.expectedDays {
				t.Errorf("days = %d, want %d", days, tt.expectedDays)
			}
			if mins != tt.expectedMins {
				t.Errorf("mins = %d, want %d", mins, tt.expectedMins)
			}
			if ticks != tt.expectedTick {
				t.Errorf("ticks = %d, want %d", ticks, tt.expectedTick)
			}
		})
	}
}

// TestRootBlockCreateBlock verifies the root block layout matches Amiga OFS spec
func TestRootBlockCreateBlock(t *testing.T) {
	rb := NewRootBlock("MyVolume", 881)
	rb.HashTable[0] = 100
	rb.HashTable[71] = 200

	block := rb.CreateBlock()

	// Verify block size
	if len(block) != adf.BlockSize {
		t.Fatalf("Block size = %d, want %d", len(block), adf.BlockSize)
	}

	// Verify type at offset 0: T_HEADER (2)
	if got := adf.GetLong(block, 0); got != adf.TypeHeader {
		t.Errorf("Type = %d, want %d (T_HEADER)", got, adf.TypeHeader)
	}

	// Verify header key at offset 1: 0 (unused for rootblock)
	if got := adf.GetLong(block, 1); got != 0 {
		t.Errorf("Header key = %d, want 0", got)
	}

	// Verify hash table size at offset 3: 72
	if got := adf.GetLong(block, 3); got != HashTableSize {
		t.Errorf("Hash table size = %d, want %d", got, HashTableSize)
	}

	// Verify hash table entries (offset 6-77)
	if got := adf.GetLong(block, 6); got != 100 {
		t.Errorf("HashTable[0] = %d, want 100", got)
	}
	if got := adf.GetLong(block, 6+71); got != 200 {
		t.Errorf("HashTable[71] = %d, want 200", got)
	}

	// Verify bitmap flag at offset -50
	if got := adf.GetLong(block, -50); got != adf.BitmapValid {
		t.Errorf("Bitmap flag = 0x%X, want 0x%X", got, adf.BitmapValid)
	}

	// Verify bitmap block pointer at offset -49
	if got := adf.GetLong(block, -49); got != 881 {
		t.Errorf("Bitmap block = %d, want 881", got)
	}

	// Verify volume name
	name := adf.GetBStr(block, adf.OffsetName, adf.MaxFilenameLen)
	if name != "MyVolume" {
		t.Errorf("Name = %s, want MyVolume", name)
	}

	// Verify secondary type at offset -1: ST_ROOT (1)
	if got := adf.GetLong(block, -1); got != adf.STRoot {
		t.Errorf("Secondary type = %d, want %d (ST_ROOT)", got, adf.STRoot)
	}

	// Verify checksum is valid
	checksum := adf.GetLong(block, 5)
	calculated := adf.CalculateChecksum(block, 5)
	if checksum != calculated {
		t.Errorf("Checksum mismatch: stored 0x%X, calculated 0x%X", checksum, calculated)
	}
}

// TestRootBlockDateFields verifies all 3 date fields are at correct offsets
func TestRootBlockDateFields(t *testing.T) {
	fixedTime := time.Date(2000, 6, 15, 14, 30, 45, 0, time.UTC)
	rb := NewRootBlock("Test", 881)
	rb.CreateTime = fixedTime
	rb.ModifyTime = fixedTime
	rb.DiskTime = fixedTime

	block := rb.CreateBlock()

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

	// Check disk date at offset -10, -9, -8
	if got := adf.GetLong(block, -10); got != days {
		t.Errorf("DiskTime days = %d, want %d", got, days)
	}
	if got := adf.GetLong(block, -9); got != mins {
		t.Errorf("DiskTime mins = %d, want %d", got, mins)
	}
	if got := adf.GetLong(block, -8); got != ticks {
		t.Errorf("DiskTime ticks = %d, want %d", got, ticks)
	}

	// Check creation date at offset -7, -6, -5
	if got := adf.GetLong(block, -7); got != days {
		t.Errorf("CreateTime days = %d, want %d", got, days)
	}
	if got := adf.GetLong(block, -6); got != mins {
		t.Errorf("CreateTime mins = %d, want %d", got, mins)
	}
	if got := adf.GetLong(block, -5); got != ticks {
		t.Errorf("CreateTime ticks = %d, want %d", got, ticks)
	}
}

// TestRootBlockAddHashEntry verifies hash chaining returns previous entry
func TestRootBlockAddHashEntry(t *testing.T) {
	rb := NewRootBlock("Test", 881)

	// Add first entry - should return 0 (no previous)
	prev := rb.AddHashEntry("testfile", 100)
	if prev != 0 {
		t.Errorf("First AddHashEntry returned %d, want 0", prev)
	}

	// Add second entry with same hash - should return previous block
	prev = rb.AddHashEntry("testfile", 200)
	if prev != 100 {
		t.Errorf("Second AddHashEntry returned %d, want 100 (previous entry)", prev)
	}

	// Verify the hash table was updated to point to new entry
	hash := HashName("testfile")
	if rb.HashTable[hash] != 200 {
		t.Errorf("HashTable[%d] = %d, want 200", hash, rb.HashTable[hash])
	}
}
