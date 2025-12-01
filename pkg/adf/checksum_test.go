package adf

import (
	"testing"
)

func TestCalculateChecksum(t *testing.T) {
	block := make([]byte, BlockSize)

	for i := range BlockSize / 4 {
		PutLong(block, i, uint32(i))
	}

	checksum := CalculateChecksum(block, OffsetChecksum)
	PutLong(block, OffsetChecksum, checksum)

	var sum uint32 = 0
	for i := range BlockSize / 4 {
		sum += GetLong(block, i)
	}
	if sum != 0 {
		t.Errorf("Checksum verification failed: sum = 0x%X, want 0", sum)
	}
}

func TestSetBlockChecksum(t *testing.T) {
	block := make([]byte, BlockSize)

	for i := range BlockSize / 4 {
		PutLong(block, i, uint32(i*0x11111111))
	}

	SetBlockChecksum(block, OffsetChecksum)

	expected := CalculateChecksum(block, OffsetChecksum)
	got := GetLong(block, OffsetChecksum)
	if got != expected {
		t.Errorf("SetBlockChecksum: got 0x%X, recalculated 0x%X", got, expected)
	}
}

func TestCalculateBootChecksum(t *testing.T) {
	bootblock := make([]byte, BlockSize*2)

	PutLong(bootblock, BootOffsetDOSType, DOSTypeOFS)
	PutLong(bootblock, BootOffsetRootPtr, RootBlock)

	SetBootChecksum(bootblock)

	checksum := CalculateBootChecksum(bootblock)
	got := GetLong(bootblock, BootOffsetChecksum)
	if got != checksum {
		t.Errorf("Boot checksum mismatch: stored 0x%X, calculated 0x%X", got, checksum)
	}
}

func TestBootChecksumKnownValue(t *testing.T) {
	bootblock := make([]byte, BlockSize*2)

	PutLong(bootblock, BootOffsetDOSType, 0x444F5300)
	PutLong(bootblock, BootOffsetRootPtr, RootBlock)

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

	checksum := GetLong(bootblock, BootOffsetChecksum)
	if checksum == 0 {
		t.Error("Boot checksum should not be zero for valid bootblock")
	}

	SetBootChecksum(bootblock)
	checksum2 := GetLong(bootblock, BootOffsetChecksum)
	if checksum != checksum2 {
		t.Errorf("Boot checksum not stable: first 0x%X, second 0x%X", checksum, checksum2)
	}
}
