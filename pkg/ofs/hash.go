package ofs

import (
	"strings"
)

// HashTableSize is the number of entries in the hash table
// For a floppy disk: (512/4) - 56 = 72
const HashTableSize = 72

// HashName calculates the Amiga filename hash
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

// IntlHashName calculates the hash using international character handling
// This is used for INTL mode disks (not needed for basic OFS)
func IntlHashName(name string) int {
	h := len(name)

	for _, c := range name {
		// International toupper
		uc := intlToUpper(c)
		h = h * 13
		h = h + int(uc)
		h = h & 0x7FF
	}

	return h % HashTableSize
}

// intlToUpper converts a character to uppercase using Amiga international rules
func intlToUpper(c rune) rune {
	if c >= 'a' && c <= 'z' {
		return c - ('a' - 'A')
	}
	// International characters (224-254, except 247 which is divide sign)
	if c >= 224 && c <= 254 && c != 247 {
		return c - 32
	}
	return c
}

