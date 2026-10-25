# gadf

Create bootable Amiga ADF disk images from executables.

This tool creates ADF (Amiga Disk File) images that boot and run a specified executable, compatible with AmigaDOS 1.2+ using the OFS (Old File System).

It was inspired by exe2adf (https://exe2adf.com/), but since that tool has no source code available, this tool was made.

Based on documentation from http://lclevy.free.fr/adflib/adf_info.html

## Features

- Creates bootable 880KB DD floppy disk images
- Cross-platform (Windows, Linux, macOS)
- Compatible with AmigaDOS 1.2+ (OFS filesystem)

## Installation

### Homebrew

On macOS, install the latest release from the Homebrew tap:

```bash
brew install --cask sphair/tap/gadf
```

### Go

```bash
go build -o gadf ./cmd/gadf
```

Or install directly:

```bash
go install github.com/sphair/gadf/cmd/gadf@latest
```

### Releases

Releases are built from `v*` Git tags using GoReleaser. Each release includes
prebuilt macOS (Apple Silicon and Intel), Linux (ARM64 and x86-64), and Windows
(x86-64) archives, along with SHA-256 checksums. A Homebrew Cask is published
to the [sphair/tap](https://github.com/sphair/homebrew-tap) tap for macOS.
Because the binary is not signed or notarized, macOS may block it; if so, clear
the quarantine attribute from the installed binary using the command shown by
the Homebrew caveat.

To enable publishing the Cask, add a `HOMEBREW_TAP_TOKEN` Actions secret to
this repository. Use a fine-grained personal access token limited to
`sphair/homebrew-tap` with Contents read/write permission. Add it under
**Settings → Secrets and variables → Actions → Repository secrets**. Once
configured, pushing a version tag (for example, `v1.0.2`) runs tests, builds
and publishes the release archives, and updates the Cask.

## Usage

```bash
# Basic usage - creates demo.adf that boots and runs demo.exe
gadf -i demo.exe

# Custom output filename
gadf -i demo.exe -a mydisk.adf

# Custom volume label
gadf -i demo.exe -l "My Demo"

# Include additional files and directories (preserves structure)
gadf -i demo.exe data/ music/

# Include directory but flatten all files to root
gadf -i demo.exe -f data/

# Mix files and directories
gadf -i demo.exe readme.txt data/ music/

# Suppress shell window on boot
gadf -i demo.exe -s
```

## Options

```
-i <file>    Input executable (required)
-a <file>    Output ADF filename (default: <input>.adf)
-l <label>   Volume label (default: input filename, max 30 chars)
-f           Flatten directory structure (put all files at root)
-s           Suppress shell window on boot
-h           Help
-version     Version info
```

You can specify multiple files and directories as arguments after the flags.
Directories are always processed recursively. Use `-f` to flatten the structure.

## Bootblock

By default, uses a minimal bootblock that works on all Kickstart versions (1.2+).

The `-s` flag adds expansion.library code to suppress the shell window flash on KS 2.0+. This option works with KS 1.2+ (the disk will boot), it just won't suppress the shell window on older Kickstarts (the expansion.library call silently fails).

## Generated Disk Structure

```
<volume label>:
├── <executable>           # Your executable
├── S/
│   └── startup-sequence   # Boot script that runs your executable
└── [additional files]     # Any additional files/directories specified
```

## License

This is free and unencumbered software released into the public domain. See [UNLICENSE](UNLICENSE) for details.
