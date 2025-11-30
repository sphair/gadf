package ofs

import (
	"testing"

	"github.com/sphair/gadf/pkg/adf"
)

// TestCreateBootBlock verifies bootblock structure and checksum
func TestCreateBootBlock(t *testing.T) {
	tests := []struct {
		name    string
		minimal bool
	}{
		{"standard", false},
		{"minimal", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bootblock := CreateBootBlock(tt.minimal)

			// Should be 1024 bytes (2 blocks)
			if len(bootblock) != adf.BlockSize*2 {
				t.Errorf("Bootblock size = %d, want %d", len(bootblock), adf.BlockSize*2)
			}

			// Check DOS type at offset 0
			dosType := adf.GetLong(bootblock, 0)
			if dosType != adf.DOSTypeOFS {
				t.Errorf("DOS type = 0x%X, want 0x%X (DOS0)", dosType, adf.DOSTypeOFS)
			}

			// Check root block pointer at offset 8
			rootPtr := adf.GetLong(bootblock, 2)
			if rootPtr != adf.RootBlock {
				t.Errorf("Root pointer = %d, want %d", rootPtr, adf.RootBlock)
			}

			// Verify checksum is valid
			checksum := adf.GetLong(bootblock, 1)
			calculated := adf.CalculateBootChecksum(bootblock)
			if checksum != calculated {
				t.Errorf("Checksum mismatch: stored 0x%X, calculated 0x%X", checksum, calculated)
			}
		})
	}
}

// TestStandardBootCodeContent verifies standard bootblock loads expansion.library
func TestStandardBootCodeContent(t *testing.T) {
	bootblock := CreateBootBlock(false)

	dosLibFound := false
	expLibFound := false

	for i := 12; i < len(bootblock)-17; i++ {
		if string(bootblock[i:i+11]) == "dos.library" {
			dosLibFound = true
		}
		if string(bootblock[i:i+17]) == "expansion.library" {
			expLibFound = true
		}
	}

	if !dosLibFound {
		t.Error("Standard bootblock should contain 'dos.library'")
	}
	if !expLibFound {
		t.Error("Standard bootblock should contain 'expansion.library'")
	}
}

// TestMinimalBootCodeContent verifies minimal bootblock does NOT load expansion.library
func TestMinimalBootCodeContent(t *testing.T) {
	bootblock := CreateBootBlock(true)

	dosLibFound := false
	expLibFound := false

	for i := 12; i < len(bootblock)-17; i++ {
		if string(bootblock[i:i+11]) == "dos.library" {
			dosLibFound = true
		}
		if string(bootblock[i:i+17]) == "expansion.library" {
			expLibFound = true
		}
	}

	if !dosLibFound {
		t.Error("Minimal bootblock should contain 'dos.library'")
	}
	if expLibFound {
		t.Error("Minimal bootblock should NOT contain 'expansion.library'")
	}
}
