# ---------------------------------------------------------------------------
# Settings
# ---------------------------------------------------------------------------

set windows-shell := ["powershell.exe", "-NoLogo", "-Command"]

out := "build"
fixtures := out / "fixtures"
bin := "gadf" + if os_family() == "windows" { ".exe" } else { "" }
devtool := "go run ./internal/devtool"
version := "1.0.1"
# Minimal Amiga hunk executable that just returns 0
exe := fixtures / "Test"

# List recipes
default:
    @just --list

# ---------------------------------------------------------------------------
# Development
# ---------------------------------------------------------------------------

# Build the gadf binary
[group('dev')]
build:
    go build -o {{bin}} ./cmd/gadf

# Build and package the Windows release
[group('dev')]
release:
    go build -trimpath -ldflags="-s -w -X main.version={{version}}" -o gadf.exe ./cmd/gadf
    Compress-Archive -Path gadf.exe -DestinationPath gadf-windows-{{version}}.zip -Force

# Run vet and tests
[group('dev')]
test:
    go vet ./...
    go test ./...

# Remove the binary and build/
[group('dev')]
clean:
    {{devtool}} clean {{out}} {{bin}}

# ---------------------------------------------------------------------------
# Test ADFs
# ---------------------------------------------------------------------------

# Create test data files
[group('adf')]
fixtures:
    {{devtool}} fixtures {{fixtures}}

# Executable only, default (minimal) bootblock
[group('adf')]
adf-minimal: build fixtures
    ./{{bin}} -i {{exe}} -a {{out}}/minimal.adf -l Minimal

# Executable only, shell-suppress bootblock
[group('adf')]
adf-suppress: build fixtures
    ./{{bin}} -i {{exe}} -a {{out}}/suppress.adf -l Suppress -s

# Large file that uses file extension blocks
[group('adf')]
adf-large: build fixtures
    ./{{bin}} -i {{exe}} -a {{out}}/large.adf -l Large {{fixtures}}/large.txt

# Nested directories, structure preserved
[group('adf')]
adf-dirs: build fixtures
    ./{{bin}} -i {{exe}} -a {{out}}/dirs.adf -l Dirs {{fixtures}}/readme.txt {{fixtures}}/data

# Nested directories, flattened to root
[group('adf')]
adf-flat: build fixtures
    ./{{bin}} -i {{exe}} -a {{out}}/flat.adf -l Flat -f {{fixtures}}/data

# Create all test ADFs in build/
[group('adf')]
adfs: adf-minimal adf-suppress adf-large adf-dirs adf-flat
