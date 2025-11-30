package adf

// CalculateChecksum calculates the standard block checksum
// Used for rootblock, file headers, data blocks, etc.
// Algorithm: sum all longs, negate the result
// The checksum position (usually offset 5) is treated as 0 during calculation
func CalculateChecksum(block []byte, checksumIndex int) uint32 {
	var sum uint32 = 0
	numLongs := len(block) / 4

	for i := 0; i < numLongs; i++ {
		if i != checksumIndex {
			sum += GetLong(block, i)
		}
	}

	return -sum // Two's complement negation
}

// CalculateBootChecksum calculates the bootblock checksum
// Algorithm: sum all longs with overflow carry, then NOT
// The checksum at offset 4 (long index 1) is treated as 0 during calculation
func CalculateBootChecksum(bootblock []byte) uint32 {
	var checksum uint32 = 0
	numLongs := len(bootblock) / 4

	for i := 0; i < numLongs; i++ {
		if i == 1 { // Skip checksum position
			continue
		}

		var d uint32
		if i*4+4 <= len(bootblock) {
			d = GetLong(bootblock, i)
		}

		// Add with overflow detection
		newSum := checksum + d
		if newSum < checksum {
			// Overflow occurred, add carry
			newSum++
		}
		checksum = newSum
	}

	return ^checksum // Bitwise NOT
}

// SetBlockChecksum calculates and sets the checksum for a standard block
func SetBlockChecksum(block []byte, checksumIndex int) {
	// Clear existing checksum
	PutLong(block, checksumIndex, 0)
	// Calculate and set new checksum
	checksum := CalculateChecksum(block, checksumIndex)
	PutLong(block, checksumIndex, checksum)
}

// SetBootChecksum calculates and sets the bootblock checksum
func SetBootChecksum(bootblock []byte) {
	// Clear existing checksum
	PutLong(bootblock, 1, 0)
	// Calculate and set new checksum
	checksum := CalculateBootChecksum(bootblock)
	PutLong(bootblock, 1, checksum)
}

