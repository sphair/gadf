package ofs

import (
	"strings"
)

// HashTableSize is the number of entries in the hash table: (512/4) - 56 = 72
const HashTableSize = 72

// HashName implements AmigaDOS filename hashing.
// Case-insensitive, returns index into directory hash table (0-71).
func HashName(name string) int {
	name = strings.ToUpper(name)
	h := len(name)

	for _, c := range name {
		h = h * 13
		h = h + int(c)
		h = h & 0x7FF
	}

	return h % HashTableSize
}
