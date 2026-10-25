package adf

import (
	"encoding/binary"
)

const (
	longSize     = 4
	longsInBlock = BlockSize / longSize
)

// Block provides utilities for working with ADF blocks
// All multi-byte values are big-endian (Motorola 68000 format)

// GetLong reads a 32-bit big-endian value from a block at the given long index
// Negative indices count from the end of the block
func GetLong(block []byte, index int) uint32 {
	if index < 0 {
		index = longsInBlock + index
	}
	offset := index * longSize
	return binary.BigEndian.Uint32(block[offset : offset+longSize])
}

// PutLong writes a 32-bit big-endian value to a block at the given long index
// Negative indices count from the end of the block
func PutLong(block []byte, index int, value uint32) {
	if index < 0 {
		index = longsInBlock + index
	}
	offset := index * longSize
	binary.BigEndian.PutUint32(block[offset:offset+longSize], value)
}

// PutBStr writes a BCPL-style string (length byte + string data)
// at the given long index with a maximum length
func PutBStr(block []byte, index int, maxLen int, s string) {
	if index < 0 {
		index = longsInBlock + index
	}
	offset := index * longSize

	if len(s) > maxLen {
		s = s[:maxLen]
	}

	block[offset] = byte(len(s))
	copy(block[offset+1:], []byte(s))
}
