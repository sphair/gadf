package ofs

import (
	"testing"
)

func TestHashName(t *testing.T) {
	tests := []struct {
		name     string
		expected int
	}{
		// These values are computed by the AmigaDOS hash algorithm
		{"S", 24},                 // S directory
		{"C", 8},                  // C directory
		{"startup-sequence", 49}, // Standard startup file
		{"Devs", 22},
		{"L", 17},
		{"Libs", 46},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := HashName(tt.name)
			if got != tt.expected {
				t.Errorf("HashName(%q) = %d, want %d", tt.name, got, tt.expected)
			}
		})
	}
}

func TestHashNameCaseInsensitive(t *testing.T) {
	// Hash should be case-insensitive
	tests := [][]string{
		{"test", "TEST", "Test", "TeSt"},
		{"Hello", "HELLO", "hello"},
		{"startup-sequence", "STARTUP-SEQUENCE", "Startup-Sequence"},
	}

	for _, group := range tests {
		expected := HashName(group[0])
		for _, name := range group[1:] {
			got := HashName(name)
			if got != expected {
				t.Errorf("HashName(%q) = %d, want %d (same as %q)", name, got, expected, group[0])
			}
		}
	}
}

func TestHashNameRange(t *testing.T) {
	// All hashes should be in valid range [0, HashTableSize)
	testNames := []string{
		"a", "ab", "abc", "test", "verylongfilename",
		"file.txt", "data.bin", "readme", "startup-sequence",
		"1234567890", "UPPERCASE", "MixedCase",
	}

	for _, name := range testNames {
		hash := HashName(name)
		if hash < 0 || hash >= HashTableSize {
			t.Errorf("HashName(%q) = %d, out of range [0, %d)", name, hash, HashTableSize)
		}
	}
}

func TestHashNameDeterministic(t *testing.T) {
	// Same input should always produce same output
	name := "test-file.exe"
	expected := HashName(name)

	for i := 0; i < 100; i++ {
		got := HashName(name)
		if got != expected {
			t.Errorf("HashName not deterministic: got %d, want %d", got, expected)
		}
	}
}

