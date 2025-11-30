package ofs

import (
	"testing"

	"github.com/sphair/gadf/pkg/adf"
)

func TestCreateBootBlock(t *testing.T) {
	tests := []struct {
		name     string
		minimal  bool
		codeSize int
	}{
		{"standard", false, 81},
		{"minimal", true, 37},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bootblock := CreateBootBlock(tt.minimal)

			// Should be 1024 bytes
			if len(bootblock) != adf.BlockSize*2 {
				t.Errorf("Bootblock size = %d, want %d", len(bootblock), adf.BlockSize*2)
			}

			// Check DOS type
			dosType := adf.GetLong(bootblock, 0)
			if dosType != adf.DOSTypeOFS {
				t.Errorf("DOS type = 0x%X, want 0x%X (DOS0)", dosType, adf.DOSTypeOFS)
			}

			// Check root block pointer
			rootPtr := adf.GetLong(bootblock, 2)
			if rootPtr != adf.RootBlock {
				t.Errorf("Root pointer = %d, want %d", rootPtr, adf.RootBlock)
			}

			// Checksum should be non-zero
			checksum := adf.GetLong(bootblock, 1)
			if checksum == 0 {
				t.Log("Warning: bootblock checksum is 0")
			}

			// Verify checksum is valid
			calculated := adf.CalculateBootChecksum(bootblock)
			if checksum != calculated {
				t.Errorf("Checksum mismatch: stored 0x%X, calculated 0x%X", checksum, calculated)
			}
		})
	}
}

func TestBootBlockDOSMarker(t *testing.T) {
	bootblock := CreateBootBlock(false)

	// First 4 bytes should be "DOS\0"
	if bootblock[0] != 'D' || bootblock[1] != 'O' || bootblock[2] != 'S' || bootblock[3] != 0 {
		t.Errorf("DOS marker = %c%c%c%d, want DOS0",
			bootblock[0], bootblock[1], bootblock[2], bootblock[3])
	}
}

func TestStandardBootCodeContent(t *testing.T) {
	// Verify the standard boot code contains expected library names
	bootblock := CreateBootBlock(false)

	// Look for "dos.library" string
	dosLibFound := false
	expLibFound := false

	for i := 12; i < len(bootblock)-12; i++ {
		if string(bootblock[i:i+11]) == "dos.library" {
			dosLibFound = true
		}
		if i < len(bootblock)-17 && string(bootblock[i:i+17]) == "expansion.library" {
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

func TestMinimalBootCodeContent(t *testing.T) {
	bootblock := CreateBootBlock(true)

	// Look for "dos.library" string
	dosLibFound := false
	expLibFound := false

	for i := 12; i < len(bootblock)-12; i++ {
		if string(bootblock[i:i+11]) == "dos.library" {
			dosLibFound = true
		}
		if i < len(bootblock)-17 && string(bootblock[i:i+17]) == "expansion.library" {
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

func TestWriteBootBlock(t *testing.T) {
	disk := adf.NewDisk()

	// Write standard bootblock
	if err := WriteBootBlock(disk, false); err != nil {
		t.Fatalf("WriteBootBlock failed: %v", err)
	}

	// Verify DOS marker at start of disk
	if disk.Data[0] != 'D' || disk.Data[1] != 'O' || disk.Data[2] != 'S' {
		t.Error("Bootblock not written correctly to disk")
	}
}

