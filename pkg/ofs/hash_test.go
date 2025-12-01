package ofs

import (
	"testing"
)

// TestHashName verifies hash values match Amiga DOS (if these change, disk won't work)
func TestHashName(t *testing.T) {
	tests := []struct {
		name     string
		expected int
	}{
		{"S", 24},
		{"C", 8},
		{"startup-sequence", 49},
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
			if got < 0 || got >= HashTableSize {
				t.Errorf("HashName(%q) = %d, out of range [0, %d)", tt.name, got, HashTableSize)
			}
		})
	}
}

// TestHashNameCaseInsensitive verifies case-insensitivity (critical for Amiga compatibility)
func TestHashNameCaseInsensitive(t *testing.T) {
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
