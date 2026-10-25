package ofs

import (
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/sphair/gadf/pkg/adf"
)

func TestGetOrCreateDirCreatesHierarchy(t *testing.T) {
	b := NewBuilder("TestDisk")
	if err := b.Build(); err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	dir, err := b.getOrCreateDir("foo\\bar/baz")
	if err != nil {
		t.Fatalf("getOrCreateDir failed: %v", err)
	}
	if dir == nil {
		t.Fatal("getOrCreateDir returned nil directory")
	}
	if dir.Name != "baz" {
		t.Fatalf("leaf directory name = %s, want baz", dir.Name)
	}

	top, ok := b.directories["foo"]
	if !ok {
		t.Fatalf("expected top-level directory foo to exist")
	}
	if top.Parent != uint32(adf.RootBlock) {
		t.Fatalf("top directory should be rooted at %d, got %d", adf.RootBlock, top.Parent)
	}

	parent, ok := b.directories["foo/bar"]
	if !ok {
		t.Fatalf("expected intermediate directory foo/bar to exist")
	}
	if parent.Parent != top.BlockNum {
		t.Fatalf("intermediate parent block = %d, want %d", parent.Parent, top.BlockNum)
	}
	if dir.Parent != parent.BlockNum {
		t.Fatalf("child parent block = %d, want %d", dir.Parent, parent.BlockNum)
	}
}

func TestAddDirectoryPreservesStructure(t *testing.T) {
	tmpDir := t.TempDir()
	source := filepath.Join(tmpDir, "src")
	if err := os.MkdirAll(filepath.Join(source, "nested", "deep"), 0o755); err != nil {
		t.Fatalf("mkdir failed: %v", err)
	}

	files := []struct {
		relPath  string
		contents string
	}{
		{"root.txt", "root"},
		{filepath.Join("nested", "child.txt"), "child"},
		{filepath.Join("nested", "deep", "leaf.txt"), "leaf"},
	}

	for _, f := range files {
		target := filepath.Join(source, f.relPath)
		if err := os.WriteFile(target, []byte(f.contents), 0o644); err != nil {
			t.Fatalf("write %s failed: %v", f.relPath, err)
		}
	}

	b := NewBuilder("TestDisk")
	if err := b.Build(); err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	added, err := b.AddDirectory(source, "assets", false)
	if err != nil {
		t.Fatalf("AddDirectory failed: %v", err)
	}

	expectedDirs := []string{
		"assets",
		"assets/nested",
		"assets/nested/deep",
	}
	for _, d := range expectedDirs {
		if _, ok := b.directories[d]; !ok {
			t.Fatalf("expected directory %s to be created", d)
		}
	}

	expectedFiles := []string{
		"assets/root.txt",
		"assets/nested/child.txt",
		"assets/nested/deep/leaf.txt",
	}
	for _, f := range expectedFiles {
		if !containsString(b.GetFileList(), f) {
			t.Fatalf("expected file list to contain %s", f)
		}
	}

	if !containsString(added, "assets/nested/") {
		t.Fatalf("expected AddDirectory to report assets/nested/ (got %v)", added)
	}
}

func TestAddFileEmptyDoesNotAllocateDataBlocks(t *testing.T) {
	b := NewBuilder("TestDisk")
	if err := b.Build(); err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	if err := b.AddFile("", "empty", []byte{}); err != nil {
		t.Fatalf("AddFile failed: %v", err)
	}

	hash := HashName("empty")
	headerBlock := b.rootBlock.HashTable[hash]
	if headerBlock == 0 {
		t.Fatalf("root hash entry for empty should not be zero")
	}

	if headerBlock >= adf.TotalBlocks {
		t.Fatalf("header block %d out of range", headerBlock)
	}
	offset := int(headerBlock) * adf.BlockSize
	block := b.disk.Data[offset : offset+adf.BlockSize]

	if got := adf.GetLong(block, adf.OffsetHighSeq); got != 0 {
		t.Fatalf("high seq = %d, want 0 for empty file", got)
	}
	if got := adf.GetLong(block, adf.OffsetFirstDataLong); got != 0 {
		t.Fatalf("first data pointer = %d, want 0", got)
	}
	if got := adf.GetLong(block, adf.OffsetFirstDataPtr); got != 0 {
		t.Fatalf("data block pointer slot should be zero, got %d", got)
	}
}

func containsString(items []string, target string) bool {
	return slices.Contains(items, target)
}
