// devtool provides cross-platform helpers for the justfile.
//
//	devtool fixtures <dir>   create test data files in dir
//	devtool clean <path>...  remove paths
package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"os"
	"path/filepath"

	"github.com/sphair/gadf/pkg/ofs"
)

// Number of file extension blocks the large test file should need.
const largeExtBlocks = 2

// AmigaDOS hunk types
const (
	hunkHeader = 0x3F3
	hunkCode   = 0x3E9
	hunkEnd    = 0x3F2
)

// 68000 opcodes
const (
	opMoveqZeroD0 = 0x7000 // moveq #0,d0
	opRts         = 0x4E75 // rts
)

func main() {
	if len(os.Args) < 3 {
		fmt.Fprintln(os.Stderr, "usage: devtool fixtures <dir> | clean <path>...")
		os.Exit(2)
	}

	var err error
	switch os.Args[1] {
	case "fixtures":
		err = writeFixtures(os.Args[2])
	case "clean":
		err = clean(os.Args[2:])
	default:
		err = fmt.Errorf("unknown command %q", os.Args[1])
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "devtool:", err)
		os.Exit(1)
	}
}

func writeFixtures(dir string) error {
	largeSize := ofs.MaxDataBlocks * (1 + largeExtBlocks) * ofs.OFSDataSize
	line := []byte("gadf\n")
	large := bytes.Repeat(line, largeSize/len(line)+1)[:largeSize]

	files := map[string][]byte{
		"Test":                minimalExecutable(),
		"readme.txt":          []byte("Hello from gadf\n"),
		"data/top.txt":        []byte("top level\n"),
		"data/sub/nested.txt": []byte("nested\n"),
		"large.txt":           large,
	}
	for name, content := range files {
		path := filepath.Join(dir, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(path, content, 0o644); err != nil {
			return err
		}
	}
	return nil
}

// minimalExecutable returns a single-hunk Amiga executable that exits with 0.
func minimalExecutable() []byte {
	var code []byte
	code = binary.BigEndian.AppendUint16(code, opMoveqZeroD0)
	code = binary.BigEndian.AppendUint16(code, opRts)
	codeLongs := uint32(len(code) / 4)

	var exe []byte
	for _, v := range []uint32{
		hunkHeader,
		0,    // no resident library names
		1,    // hunk table size
		0, 0, // first and last hunk
		codeLongs, // size of hunk 0
		hunkCode,
		codeLongs,
	} {
		exe = binary.BigEndian.AppendUint32(exe, v)
	}
	exe = append(exe, code...)
	return binary.BigEndian.AppendUint32(exe, hunkEnd)
}

func clean(paths []string) error {
	for _, p := range paths {
		if err := os.RemoveAll(p); err != nil {
			return err
		}
	}
	return nil
}
