package ofs

import (
	"testing"

	"github.com/sphair/gadf/pkg/adf"
)

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

			if len(bootblock) != adf.BlockSize*2 {
				t.Errorf("Bootblock size = %d, want %d", len(bootblock), adf.BlockSize*2)
			}

			dosType := adf.GetLong(bootblock, adf.BootOffsetDOSType)
			if dosType != adf.DOSTypeOFS {
				t.Errorf("DOS type = 0x%X, want 0x%X (DOS0)", dosType, adf.DOSTypeOFS)
			}

			rootPtr := adf.GetLong(bootblock, adf.BootOffsetRootPtr)
			if rootPtr != adf.RootBlock {
				t.Errorf("Root pointer = %d, want %d", rootPtr, adf.RootBlock)
			}

			checksum := adf.GetLong(bootblock, adf.BootOffsetChecksum)
			calculated := adf.CalculateBootChecksum(bootblock)
			if checksum != calculated {
				t.Errorf("Checksum mismatch: stored 0x%X, calculated 0x%X", checksum, calculated)
			}
		})
	}
}
