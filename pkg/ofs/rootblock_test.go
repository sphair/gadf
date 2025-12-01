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

func TestRootBlockCreateBlock(t *testing.T) {
	rb := NewRootBlock("MyVolume", 881)
	rb.HashTable[0] = 100
	rb.HashTable[71] = 200

	block := rb.CreateBlock()

	if len(block) != adf.BlockSize {
		t.Fatalf("Block size = %d, want %d", len(block), adf.BlockSize)
	}

	if got := adf.GetLong(block, adf.OffsetBlockType); got != adf.TypeHeader {
		t.Errorf("Type = %d, want %d (T_HEADER)", got, adf.TypeHeader)
	}
	if got := adf.GetLong(block, adf.OffsetHeaderKey); got != 0 {
		t.Errorf("Header key = %d, want 0", got)
	}
	if got := adf.GetLong(block, adf.OffsetHashTableSize); got != HashTableSize {
		t.Errorf("Hash table size = %d, want %d", got, HashTableSize)
	}
	if got := adf.GetLong(block, adf.OffsetHashTable); got != 100 {
		t.Errorf("HashTable[0] = %d, want 100", got)
	}
	if got := adf.GetLong(block, adf.OffsetHashTable+71); got != 200 {
		t.Errorf("HashTable[71] = %d, want 200", got)
	}
	if got := adf.GetLong(block, adf.OffsetBitmapFlag); got != adf.BitmapValid {
		t.Errorf("Bitmap flag = 0x%X, want 0x%X", got, adf.BitmapValid)
	}
	if got := adf.GetLong(block, adf.OffsetBitmapPages); got != 881 {
		t.Errorf("Bitmap block = %d, want 881", got)
	}

	name := adf.GetBStr(block, adf.OffsetName, adf.MaxFilenameLen)
	if name != "MyVolume" {
		t.Errorf("Name = %s, want MyVolume", name)
	}

	if got := adf.GetLong(block, adf.OffsetSecondaryType); got != adf.STRoot {
		t.Errorf("Secondary type = %d, want %d (ST_ROOT)", got, adf.STRoot)
	}

	checksum := adf.GetLong(block, adf.OffsetChecksum)
	calculated := adf.CalculateChecksum(block, adf.OffsetChecksum)
	if checksum != calculated {
		t.Errorf("Checksum mismatch: stored 0x%X, calculated 0x%X", checksum, calculated)
	}
}

func TestRootBlockDateFields(t *testing.T) {
	fixedTime := time.Date(2000, 6, 15, 14, 30, 45, 0, time.UTC)
	rb := NewRootBlock("Test", 881)
	rb.CreateTime = fixedTime
	rb.ModifyTime = fixedTime
	rb.DiskTime = fixedTime

	block := rb.CreateBlock()

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

	if got := adf.GetLong(block, adf.OffsetDiskDateDays); got != days {
		t.Errorf("DiskTime days = %d, want %d", got, days)
	}
	if got := adf.GetLong(block, adf.OffsetDiskDateMins); got != mins {
		t.Errorf("DiskTime mins = %d, want %d", got, mins)
	}
	if got := adf.GetLong(block, adf.OffsetDiskDateTicks); got != ticks {
		t.Errorf("DiskTime ticks = %d, want %d", got, ticks)
	}

	if got := adf.GetLong(block, adf.OffsetCreateDateDays); got != days {
		t.Errorf("CreateTime days = %d, want %d", got, days)
	}
	if got := adf.GetLong(block, adf.OffsetCreateDateMins); got != mins {
		t.Errorf("CreateTime mins = %d, want %d", got, mins)
	}
	if got := adf.GetLong(block, adf.OffsetCreateDateTicks); got != ticks {
		t.Errorf("CreateTime ticks = %d, want %d", got, ticks)
	}
}

func TestRootBlockAddHashEntry(t *testing.T) {
	rb := NewRootBlock("Test", 881)

	prev := rb.AddHashEntry("testfile", 100)
	if prev != 0 {
		t.Errorf("First AddHashEntry returned %d, want 0", prev)
	}

	prev = rb.AddHashEntry("testfile", 200)
	if prev != 100 {
		t.Errorf("Second AddHashEntry returned %d, want 100 (previous entry)", prev)
	}

	hash := HashName("testfile")
	if rb.HashTable[hash] != 200 {
		t.Errorf("HashTable[%d] = %d, want 200", hash, rb.HashTable[hash])
	}
}
