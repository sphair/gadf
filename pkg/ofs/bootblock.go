package ofs

import (
	"github.com/sphair/gadf/pkg/adf"
)

// StandardBootCode is the AmigaDOS 3.0 boot code with shell-suppress.
// Sets expansion.library bit 6 to suppress CLI window on KS 2.0+.
// Works on all Kickstart versions; shell-suppress only affects 2.0+.
var StandardBootCode = []byte{
	0x43, 0xFA, 0x00, 0x3E, // lea     exp(pc),a1
	0x70, 0x25, // moveq   #37,d0
	0x4E, 0xAE, 0xFD, 0xD8, // jsr     -552(a6)        ; OpenLibrary()
	0x4A, 0x80, // tst.l   d0
	0x67, 0x0C, // beq.b   findDos
	0x22, 0x40, // move.l  d0,a1
	0x08, 0xE9, 0x00, 0x06, // bset    #6,34(a1)       ; EBB_SILENTSTART
	0x00, 0x22, //
	0x4E, 0xAE, 0xFE, 0x62, // jsr     -414(a6)        ; CloseLibrary()
	0x43, 0xFA, 0x00, 0x18, // lea     dos(pc),a1
	0x4E, 0xAE, 0xFF, 0xA0, // jsr     -96(a6)         ; FindResident()
	0x4A, 0x80, // tst.l   d0
	0x67, 0x0A, // beq.b   error
	0x20, 0x40, // move.l  d0,a0
	0x20, 0x68, 0x00, 0x16, // move.l  22(a0),a0       ; RT_INIT
	0x70, 0x00, // moveq   #0,d0
	0x4E, 0x75, // rts
	0x70, 0xFF, // moveq   #-1,d0          ; error
	0x4E, 0x75, // rts
	0x64, 0x6F, 0x73, 0x2E, 0x6C, 0x69, 0x62, 0x72, 0x61, 0x72, 0x79, 0x00, // "dos.library"
	0x65, 0x78, 0x70, 0x61, 0x6E, 0x73, 0x69, 0x6F, 0x6E, 0x2E, 0x6C, 0x69, 0x62, 0x72, 0x61, 0x72, 0x79, 0x00, // "expansion.library"
}

// MinimalBootCode is the standard AmigaDOS boot code with RT_INIT check.
// FindResident("dos.library") -> RT_INIT -> return.
// Works on all Kickstart versions (1.2+).
var MinimalBootCode = []byte{
	// String at offset 30, PC at 2, displacement = 28 = 0x1C
	0x43, 0xFA, 0x00, 0x1C, // lea     dos(pc),a1
	0x4E, 0xAE, 0xFF, 0xA0, // jsr     -96(a6)         ; FindResident()
	0x4A, 0x80, // tst.l   d0
	0x67, 0x0E, // beq.b   error
	0x20, 0x40, // move.l  d0,a0
	0x20, 0x28, 0x00, 0x16, // move.l  22(a0),d0       ; RT_INIT to d0
	0x67, 0x06, // beq.b   error           ; null check
	0x20, 0x40, // move.l  d0,a0
	0x70, 0x00, // moveq   #0,d0
	0x4E, 0x75, // rts
	0x70, 0xFF, // moveq   #-1,d0          ; error
	0x4E, 0x75, // rts
	0x64, 0x6F, 0x73, 0x2E, 0x6C, 0x69, 0x62, 0x72, 0x61, 0x72, 0x79, 0x00, // "dos.library"
}

// CreateBootBlock creates a bootable bootblock for OFS
// If minimal is true, uses the smaller boot code without expansion.library
func CreateBootBlock(minimal bool) []byte {
	bootblock := make([]byte, adf.BootBlockSize)

	adf.PutLong(bootblock, adf.BootOffsetDOSType, adf.DOSTypeOFS)
	adf.PutLong(bootblock, adf.BootOffsetRootPtr, adf.RootBlock)

	if minimal {
		copy(bootblock[adf.BootCodeOffset:], MinimalBootCode)
	} else {
		copy(bootblock[adf.BootCodeOffset:], StandardBootCode)
	}

	adf.SetBootChecksum(bootblock)

	return bootblock
}

// WriteBootBlock writes the bootblock to the disk
// If minimal is true, uses the smaller boot code without expansion.library
func WriteBootBlock(disk *adf.Disk, minimal bool) error {
	bootblock := CreateBootBlock(minimal)
	return disk.WriteBootBlock(bootblock)
}
