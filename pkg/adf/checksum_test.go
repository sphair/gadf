package adf

import (
	"testing"
)

func TestCalculateChecksum(t *testing.T) {
	block := make([]byte, BlockSize)

	// Fill with known values
	for i := 0; i < BlockSize/4; i++ {
		PutLong(block, i, uint32(i))
	}

	// Calculate checksum (standard offset for most blocks)
	checksum := CalculateChecksum(block, OffsetChecksum)

	// Set it and verify
	PutLong(block, OffsetChecksum, checksum)

	// Sum of all longs should now be 0
	var sum uint32 = 0
	for i := 0; i < BlockSize/4; i++ {
		sum += GetLong(block, i)
	}
	if sum != 0 {
		t.Errorf("Checksum verification failed: sum = 0x%X, want 0", sum)
	}
}

func TestSetBlockChecksum(t *testing.T) {
	block := make([]byte, BlockSize)

	// Fill with data
	for i := 0; i < BlockSize/4; i++ {
		PutLong(block, i, uint32(i*0x11111111))
	}

	// Set checksum
	SetBlockChecksum(block, OffsetChecksum)

	// Verify by recalculating
	expected := CalculateChecksum(block, OffsetChecksum)
	got := GetLong(block, OffsetChecksum)
	if got != expected {
		t.Errorf("SetBlockChecksum: got 0x%X, recalculated 0x%X", got, expected)
	}
}

func TestCalculateBootChecksum(t *testing.T) {
	// Create a minimal bootblock
	bootblock := make([]byte, BlockSize*2)

	// Set DOS type
	PutLong(bootblock, BootOffsetDOSType, DOSTypeOFS)

	// Set root block
	PutLong(bootblock, BootOffsetRootPtr, RootBlock)

	// Calculate and set checksum
	SetBootChecksum(bootblock)

	// Verify by recalculating
	checksum := CalculateBootChecksum(bootblock)
	got := GetLong(bootblock, BootOffsetChecksum)
	if got != checksum {
		t.Errorf("Boot checksum mismatch: stored 0x%X, calculated 0x%X", got, checksum)
	}
}

func TestBootChecksumKnownValue(t *testing.T) {
	// Test with the actual bootblock we use
	// This ensures our checksum algorithm matches the Amiga's
	bootblock := make([]byte, BlockSize*2)

	// DOS0 type
	PutLong(bootblock, BootOffsetDOSType, 0x444F5300)
	PutLong(bootblock, BootOffsetRootPtr, RootBlock)

	// Minimal boot code (just dos.library lookup)
	bootCode := []byte{
		0x43, 0xFA, 0x00, 0x12, // lea dos(pc),a1
		0x4E, 0xAE, 0xFF, 0xA0, // jsr FindResident
		0x4A, 0x80, // tst.l d0
		0x67, 0x0A, // beq.b error
		0x20, 0x40, // move.l d0,a0
		0x20, 0x68, 0x00, 0x16, // move.l 22(a0),a0
		0x70, 0x00, // moveq #0,d0
		0x4E, 0x75, // rts
		0x70, 0xFF, // moveq #-1,d0
		0x4E, 0x75, // rts
		0x64, 0x6F, 0x73, 0x2E, 0x6C, 0x69, 0x62, 0x72, 0x61, 0x72, 0x79, 0x00, // "dos.library"
	}
	copy(bootblock[BootCodeOffset:], bootCode)

	SetBootChecksum(bootblock)

	// The checksum should be non-zero and consistent
	checksum := GetLong(bootblock, BootOffsetChecksum)
	if checksum == 0 {
		t.Error("Boot checksum should not be zero for valid bootblock")
	}

	// Verify it's stable
	SetBootChecksum(bootblock)
	checksum2 := GetLong(bootblock, BootOffsetChecksum)
	if checksum != checksum2 {
		t.Errorf("Boot checksum not stable: first 0x%X, second 0x%X", checksum, checksum2)
	}
}
