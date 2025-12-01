package adf

import (
	"encoding/binary"
)

const (
	longSize     = 4
	wordSize     = 2
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

// GetWord reads a 16-bit big-endian value from a block at the given byte offset
func GetWord(block []byte, offset int) uint16 {
	return binary.BigEndian.Uint16(block[offset : offset+wordSize])
}

// PutWord writes a 16-bit big-endian value to a block at the given byte offset
func PutWord(block []byte, offset int, value uint16) {
	binary.BigEndian.PutUint16(block[offset:offset+wordSize], value)
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

// GetBStr reads a BCPL-style string from the given long index
func GetBStr(block []byte, index int, maxLen int) string {
	if index < 0 {
		index = longsInBlock + index
	}
	offset := index * longSize

	length := min(int(block[offset]), maxLen)

	return string(block[offset+1 : offset+1+length])
}
