// gadf - Create bootable Amiga ADF disk images from executables
//
// This tool creates ADF (Amiga Disk File) images that boot and run
// a specified executable, compatible with AmigaDOS 1.2+ (OFS filesystem).
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"runtime/debug"
	"slices"
	"strings"

	"github.com/sphair/gadf/pkg/adf"
	"github.com/sphair/gadf/pkg/ofs"
)

var version = "dev"

func versionString() string {
	if version != "dev" {
		return version
	}

	info, ok := debug.ReadBuildInfo()
	if ok && info.Main.Version != "" && info.Main.Version != "(devel)" {
		return info.Main.Version
	}
	return version
}

func main() {
	inputFile := flag.String("i", "", "Input executable file (required)")
	outputFile := flag.String("a", "", "Output ADF filename (default: <input>.adf)")
	label := flag.String("l", "", "Volume label (default: input filename, max 30 chars)")
	flatten := flag.Bool("f", false, "Flatten directory structure (recurse but put all files at root)")
	shellSuppress := flag.Bool("s", false, "Suppress shell window on boot (KS 2.0+ only, uses expansion.library)")
	showVersion := flag.Bool("version", false, "Show version information")
	showHelp := flag.Bool("h", false, "Show help")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "gadf v%s - Create bootable Amiga ADF disk images\n\n", versionString())
		fmt.Fprintf(os.Stderr, "Usage: gadf -i <executable> [options] [files/directories...]\n\n")
		fmt.Fprintf(os.Stderr, "Options:\n")
		fmt.Fprintf(os.Stderr, "  -i <file>      Input executable file (required)\n")
		fmt.Fprintf(os.Stderr, "  -a <file>      Output ADF filename (default: <input>.adf)\n")
		fmt.Fprintf(os.Stderr, "  -l <label>     Volume label (default: input filename, max 30 chars)\n")
		fmt.Fprintf(os.Stderr, "  -f             Flatten directory structure (put all files at root)\n")
		fmt.Fprintf(os.Stderr, "  -s             Suppress shell window on boot\n")
		fmt.Fprintf(os.Stderr, "  -h             Show this help\n")
		fmt.Fprintf(os.Stderr, "  -version       Show version information\n")
		fmt.Fprintf(os.Stderr, "\nAdditional files/directories can be specified as arguments after the flags.\n")
		fmt.Fprintf(os.Stderr, "Directories are always processed recursively.\n")
		fmt.Fprintf(os.Stderr, "\nBootblock modes:\n")
		fmt.Fprintf(os.Stderr, "  Default        Simple boot code, works on all Kickstart versions (1.2+)\n")
		fmt.Fprintf(os.Stderr, "  -s             Uses expansion.library to suppress shell window (KS 2.0+)\n")
		fmt.Fprintf(os.Stderr, "                 This option works with KS 1.2+, it just won't suppress the shell window\n")
		fmt.Fprintf(os.Stderr, "\nThe generated ADF will boot and run the executable on AmigaDOS 1.2+\n")
		fmt.Fprintf(os.Stderr, "\nExamples:\n")
		fmt.Fprintf(os.Stderr, "  gadf -i demo.exe                          # Creates demo.adf\n")
		fmt.Fprintf(os.Stderr, "  gadf -i demo.exe -a mydisk.adf            # Custom output name\n")
		fmt.Fprintf(os.Stderr, "  gadf -i demo.exe -l \"My Demo\"             # Custom volume label\n")
		fmt.Fprintf(os.Stderr, "  gadf -i demo.exe data/ music/             # Include directories\n")
		fmt.Fprintf(os.Stderr, "  gadf -i demo.exe -f data/                 # Include directory (flatten to root)\n")
		fmt.Fprintf(os.Stderr, "  gadf -i demo.exe readme.txt data/ img/    # Mix files and directories\n")
		fmt.Fprintf(os.Stderr, "  gadf -i demo.exe -s                       # Suppress shell on KS 2.0+\n")
	}

	flag.Parse()

	if *showVersion {
		fmt.Printf("gadf v%s\n", versionString())
		fmt.Println("Create bootable Amiga ADF disk images from executables")
		fmt.Println("Compatible with AmigaDOS 1.2+ (OFS filesystem)")
		os.Exit(0)
	}

	if *showHelp {
		flag.Usage()
		os.Exit(0)
	}

	if *inputFile == "" {
		fmt.Fprintln(os.Stderr, "Error: input file (-i) is required")
		fmt.Fprintln(os.Stderr, "")
		flag.Usage()
		os.Exit(1)
	}

	if _, err := os.Stat(*inputFile); os.IsNotExist(err) {
		fmt.Fprintf(os.Stderr, "Error: input file does not exist: %s\n", *inputFile)
		os.Exit(1)
	}

	output := *outputFile
	if output == "" {
		base := filepath.Base(*inputFile)
		ext := filepath.Ext(base)
		if ext != "" {
			base = strings.TrimSuffix(base, ext)
		}
		output = base + ".adf"
	}

	volumeLabel := *label
	if volumeLabel == "" {
		base := filepath.Base(*inputFile)
		ext := filepath.Ext(base)
		if ext != "" {
			base = strings.TrimSuffix(base, ext)
		}
		volumeLabel = base
	}

	if len(volumeLabel) > 30 {
		volumeLabel = volumeLabel[:30]
		fmt.Printf("Warning: volume label truncated to 30 characters: %s\n", volumeLabel)
	}

	if err := ofs.ValidateFilename(volumeLabel); err != nil {
		fmt.Fprintf(os.Stderr, "Error: invalid volume label: %v\n", err)
		os.Exit(1)
	}

	execName := filepath.Base(*inputFile)
	if err := ofs.ValidateFilename(execName); err != nil {
		fmt.Fprintf(os.Stderr, "Error: invalid executable filename: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Creating ADF: %s\n", output)
	fmt.Printf("  Volume label: %s\n", volumeLabel)
	fmt.Printf("  Executable: %s\n", execName)
	if *shellSuppress {
		fmt.Println("  Bootblock: with shell-suppress")
	} else {
		fmt.Println("  Bootblock: standard")
	}

	builder := ofs.NewBuilder(volumeLabel)
	builder.SetMinimalBoot(!*shellSuppress)

	if err := builder.Build(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: failed to initialize disk: %v\n", err)
		os.Exit(1)
	}

	if err := builder.AddFileFromDisk("", *inputFile); err != nil {
		fmt.Fprintf(os.Stderr, "Error: failed to add executable: %v\n", err)
		os.Exit(1)
	}

	additionalItems := flag.Args()
	for _, item := range additionalItems {
		info, err := os.Stat(item)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: cannot access '%s': %v\n", item, err)
			os.Exit(1)
		}

		if info.IsDir() {
			destDir := ""
			if !*flatten {
				destDir = filepath.Base(item)
			}

			if _, err := builder.AddDirectory(item, destDir, *flatten); err != nil {
				fmt.Fprintf(os.Stderr, "Error: failed to add directory '%s': %v\n", item, err)
				os.Exit(1)
			}
		} else {
			if err := builder.AddFileFromDisk("", item); err != nil {
				fmt.Fprintf(os.Stderr, "Error: failed to add file '%s': %v\n", item, err)
				os.Exit(1)
			}
		}
	}

	if err := builder.CreateStartupSequence(execName); err != nil {
		fmt.Fprintf(os.Stderr, "Error: failed to create startup-sequence: %v\n", err)
		os.Exit(1)
	}

	if err := builder.Finalize(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: failed to finalize disk: %v\n", err)
		os.Exit(1)
	}

	if err := builder.Save(output); err != nil {
		fmt.Fprintf(os.Stderr, "Error: failed to save ADF: %v\n", err)
		os.Exit(1)
	}

	totalBlocks := adf.TotalBlocks
	freeBlocks := builder.GetFreeBlocks()
	usedBlocks := totalBlocks - freeBlocks
	usedPercent := (usedBlocks * 100) / totalBlocks

	fmt.Printf("\nSuccess! %d blocks used (%d%% of total %d, %d free)\n\n", usedBlocks, usedPercent, totalBlocks, freeBlocks)

	fmt.Println("ADF structure:")

	dirs := builder.GetDirectories()
	files := builder.GetFileList()

	filesByDir := make(map[string][]string)
	rootFiles := []string{}

	for _, file := range files {
		dir := filepath.Dir(file)
		if dir == "." {
			rootFiles = append(rootFiles, filepath.Base(file))
		} else {
			filesByDir[dir] = append(filesByDir[dir], filepath.Base(file))
		}
	}

	slices.Sort(dirs)
	slices.Sort(rootFiles)

	for _, file := range rootFiles {
		fmt.Printf("  %s\n", file)
	}

	for _, dir := range dirs {
		fmt.Printf("  %s/\n", dir)
		dirFiles := filesByDir[dir]
		slices.Sort(dirFiles)
		for _, file := range dirFiles {
			fmt.Printf("    %s\n", file)
		}
	}
}
