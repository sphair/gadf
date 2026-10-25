package adf

import (
	"testing"
)

func TestGetPutLong(t *testing.T) {
	block := make([]byte, BlockSize)

	PutLong(block, 0, 0x12345678)
	if got := GetLong(block, 0); got != 0x12345678 {
		t.Errorf("GetLong(0) = 0x%X, want 0x12345678", got)
	}

	if block[0] != 0x12 || block[1] != 0x34 || block[2] != 0x56 || block[3] != 0x78 {
		t.Errorf("Big-endian order wrong: got %X %X %X %X", block[0], block[1], block[2], block[3])
	}

	PutLong(block, -1, 0xDEADBEEF)
	if got := GetLong(block, -1); got != 0xDEADBEEF {
		t.Errorf("GetLong(-1) = 0x%X, want 0xDEADBEEF", got)
	}

	if got := GetLong(block, 127); got != 0xDEADBEEF {
		t.Errorf("GetLong(127) = 0x%X, want 0xDEADBEEF (same as -1)", got)
	}
}

func TestPutBStr(t *testing.T) {
	block := make([]byte, BlockSize)

	PutBStr(block, 0, 30, "Hello")
	if got := string(block[:6]); got != "\x05Hello" {
		t.Errorf("BCPL string = %q, want %q", got, "\x05Hello")
	}

	longStr := "ThisIsAVeryLongStringThatExceedsTheMaximumLength"
	PutBStr(block, 10, 10, longStr)
	if got := string(block[40:51]); got != "\x0aThisIsAVer" {
		t.Errorf("Truncated BCPL string = %q, want %q", got, "\x0aThisIsAVer")
	}

	PutBStr(block, -20, 30, "AtEnd")
	if got := string(block[BlockSize-80 : BlockSize-74]); got != "\x05AtEnd" {
		t.Errorf("BCPL string at negative index = %q, want %q", got, "\x05AtEnd")
	}
}
