// Package adf provides low-level ADF disk image handling
package adf

const (
	// Disk geometry for DD (Double Density) floppy
	Cylinders       = 80
	Heads           = 2
	SectorsPerTrack = 11
	BlockSize       = 512
	BootBlockSize   = BlockSize * 2 // 1024 bytes

	// Calculated values
	TotalBlocks = Cylinders * Heads * SectorsPerTrack // 1760 blocks
	DiskSize    = TotalBlocks * BlockSize             // 901120 bytes

	// Key block positions
	BootBlockStart = 0
	BootBlockEnd   = 1
	ReservedBlocks = 2   // de_Reserved: blocks not tracked by the bitmap
	RootBlock      = 880 // Middle of disk
	BitmapBlock    = 881 // First bitmap block

	// DOS types (bootblock identifier)
	DOSTypeOFS = 0x444F5300 // "DOS\0" - Old File System
	DOSTypeFFS = 0x444F5301 // "DOS\1" - Fast File System

	// Block types
	TypeHeader   = 2  // T_SHORT
	TypeData     = 8  // T_DATA
	TypeList     = 16 // T_LIST (file extension)
	TypeDirCache = 33 // T_DIR_CACHE

	// Secondary types (stored at end of block)
	STRoot    = 1          // Root block
	STUserDir = 2          // User directory
	STFile    = 0xFFFFFFFD // -3 as uint32, file header

	// Bitmap
	BitmapValid         = 0xFFFFFFFF // All bits set = valid/free
	BitmapChecksumLong  = 0          // Bitmap checksum long index
	BitmapDataStartLong = 1          // First long of bitmap data (checksum at 0)

	// Limits
	MaxFilenameLen = 30

	// Block layout offsets (as long indices, negative from end)
	OffsetBlockType       = 0   // Block type identifier (T_HEADER, etc.)
	OffsetHeaderKey       = 1   // Self pointer / header key
	OffsetHighSeq         = 2   // High sequence / data count
	OffsetHashTableSize   = 3   // Hash table size (root blocks) / data size (data blocks)
	OffsetFirstDataLong   = 4   // First data pointer (standard blocks)
	OffsetChecksum        = 5   // Checksum position in most blocks
	OffsetHashTable       = 6   // Hash table start
	OffsetFirstDataPtr    = -51 // First data block pointer (file header)
	OffsetBitmapFlag      = -50 // Bitmap valid flag
	OffsetBitmapPages     = -49 // Bitmap block pointers
	OffsetProtection      = -48 // Protection bits
	OffsetFileSize        = -47 // File size in bytes
	OffsetBitmapExtension = -24 // Bitmap extension block pointer
	OffsetDateDays        = -23 // Modification date (days)
	OffsetDateMins        = -22 // Modification date (minutes)
	OffsetDateTicks       = -21 // Modification date (ticks)
	OffsetName            = -20 // BCPL name string
	OffsetDiskDateDays    = -10 // Disk alteration date (days)
	OffsetDiskDateMins    = -9  // Disk alteration date (minutes)
	OffsetDiskDateTicks   = -8  // Disk alteration date (ticks)
	OffsetCreateDateDays  = -7  // Creation date (days)
	OffsetCreateDateMins  = -6  // Creation date (minutes)
	OffsetCreateDateTicks = -5  // Creation date (ticks)
	OffsetNextHash        = -4  // Next hash entry pointer
	OffsetParent          = -3  // Parent directory pointer
	OffsetExtension       = -2  // Extension block pointer
	OffsetSecondaryType   = -1  // Secondary type identifier

	// Bootblock layout (long indices)
	BootOffsetDOSType  = 0 // DOS type identifier
	BootOffsetChecksum = 1 // Bootblock checksum field
	BootOffsetRootPtr  = 2 // Root block pointer

	// Boot code starts at byte offset 12
	BootCodeOffset = 12
)
