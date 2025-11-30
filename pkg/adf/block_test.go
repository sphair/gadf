package adf

import (
	"testing"
)

func TestGetPutLong(t *testing.T) {
	block := make([]byte, BlockSize)

	// Test positive index
	PutLong(block, 0, 0x12345678)
	if got := GetLong(block, 0); got != 0x12345678 {
		t.Errorf("GetLong(0) = 0x%X, want 0x12345678", got)
	}

	// Test big-endian byte order
	if block[0] != 0x12 || block[1] != 0x34 || block[2] != 0x56 || block[3] != 0x78 {
		t.Errorf("Big-endian order wrong: got %X %X %X %X", block[0], block[1], block[2], block[3])
	}

	// Test negative index (from end of block)
	PutLong(block, -1, 0xDEADBEEF)
	if got := GetLong(block, -1); got != 0xDEADBEEF {
		t.Errorf("GetLong(-1) = 0x%X, want 0xDEADBEEF", got)
	}

	// Verify it's at the last 4 bytes
	if got := GetLong(block, 127); got != 0xDEADBEEF {
		t.Errorf("GetLong(127) = 0x%X, want 0xDEADBEEF (same as -1)", got)
	}
}

func TestGetPutWord(t *testing.T) {
	block := make([]byte, BlockSize)

	PutWord(block, 0, 0xABCD)
	if got := GetWord(block, 0); got != 0xABCD {
		t.Errorf("GetWord(0) = 0x%X, want 0xABCD", got)
	}

	// Test big-endian
	if block[0] != 0xAB || block[1] != 0xCD {
		t.Errorf("Big-endian order wrong: got %X %X", block[0], block[1])
	}
}

func TestPutGetBStr(t *testing.T) {
	block := make([]byte, BlockSize)

	// Test normal string
	PutBStr(block, 0, 30, "Hello")
	got := GetBStr(block, 0, 30)
	if got != "Hello" {
		t.Errorf("GetBStr = %q, want %q", got, "Hello")
	}

	// Verify BCPL format (length byte + string)
	if block[0] != 5 {
		t.Errorf("Length byte = %d, want 5", block[0])
	}

	// Test truncation
	longStr := "ThisIsAVeryLongStringThatExceedsTheMaximumLength"
	PutBStr(block, 10, 10, longStr)
	got = GetBStr(block, 10, 10)
	if got != "ThisIsAVer" {
		t.Errorf("Truncated GetBStr = %q, want %q", got, "ThisIsAVer")
	}

	// Test negative index
	PutBStr(block, -20, 30, "AtEnd")
	got = GetBStr(block, -20, 30)
	if got != "AtEnd" {
		t.Errorf("GetBStr(-20) = %q, want %q", got, "AtEnd")
	}
}

