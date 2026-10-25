package adf

import (
	"fmt"
	"os"
)

// Disk represents an ADF disk image
type Disk struct {
	Data [DiskSize]byte
}

// NewDisk creates a new empty ADF disk
func NewDisk() *Disk {
	return &Disk{}
}

// WriteBlock writes a 512-byte block to the disk
func (d *Disk) WriteBlock(blockNum int, data []byte) error {
	if blockNum < 0 || blockNum >= TotalBlocks {
		return fmt.Errorf("block number %d out of range (0-%d)", blockNum, TotalBlocks-1)
	}
	if len(data) != BlockSize {
		return fmt.Errorf("block data must be exactly %d bytes, got %d", BlockSize, len(data))
	}
	offset := blockNum * BlockSize
	copy(d.Data[offset:offset+BlockSize], data)
	return nil
}

// WriteBootBlock writes data to the bootblock area (blocks 0-1, 1024 bytes)
func (d *Disk) WriteBootBlock(data []byte) error {
	if len(data) > BlockSize*2 {
		return fmt.Errorf("bootblock data too large: %d bytes (max 1024)", len(data))
	}
	copy(d.Data[0:], data)
	return nil
}

// Save writes the disk image to a file
func (d *Disk) Save(filename string) error {
	return os.WriteFile(filename, d.Data[:], 0644)
}
