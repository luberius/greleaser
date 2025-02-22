# GReleaser

GReleaser is a Go-based tool for automating GitHub releases. It builds your project, creates a ZIP archive of the build output, and publishes it as a GitHub release with an automatically generated changelog.

## Features

- Automated build process with configurable build commands
- ZIP archive creation of build artifacts
- Automatic changelog generation from git commits
- GitHub release creation with asset upload
- Cross-platform build support (Linux, macOS, Windows)
- Optimized binary sizes

## Prerequisites

- Go 1.16 or later
- Git installed and configured
- GitHub personal access token with `repo` scope
- Build tools for your project (e.g., npm, yarn, etc.)

## Installation

```bash
# Clone the repository
git clone https://github.com/yourusername/greleaser.git
cd greleaser

# Initialize Go module
go mod init greleaser
```

## Configuration

Create a `.release.env` file in your project root with the following required configuration:

```env
GITHUB_TOKEN=your-github-token-here
BUILD_PATH=dist
BUILD_COMMAND=npm run build
```

You can also set these as environment variables:

```bash
export GITHUB_TOKEN=your-github-token-here
export BUILD_PATH=dist
export BUILD_COMMAND="npm run build"
```

### Configuration Options

- `GITHUB_TOKEN`: Your GitHub personal access token (required)
- `BUILD_PATH`: Path to the directory containing build artifacts (required)
- `BUILD_COMMAND`: Command to build your project (required)

## Usage

### Basic Usage

```bash
# Create a new release
go run main.go v1.0.0
```

### Building Optimized Binaries

The project includes scripts for building optimized binaries for multiple platforms.

```bash
# Build for your current platform
make build-local

# Build for all supported platforms
make build-all

# Clean build artifacts
make clean
```

Built binaries will be placed in the `bin/` directory with platform-specific names.

### Using UPX Compression (Optional)

For even smaller binary sizes, you can install UPX:

```bash
# macOS
brew install upx

# Ubuntu/Debian
sudo apt-get install upx

# Windows (with chocolatey)
choco install upx
```

If UPX is installed, the build script will automatically compress the binaries.

## Binary Sizes

Approximate binary sizes:
- Default build: ~7-8MB
- With optimization flags: ~5-6MB
- With UPX compression: ~2-3MB

## Project Structure

```
greleaser/
├── main.go           # Main application code
├── go.mod           # Go module file
├── .release.env     # Configuration file
├── build.sh         # Build script for multiple platforms
└── Makefile         # Make targets for building
```

## Error Handling

The tool includes comprehensive error handling for:
- Missing configuration
- Build failures
- ZIP creation issues
- GitHub API errors
- File system operations

## Contributing

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add some amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## License

This project is licensed under the MIT License

## Acknowledgments

- Built with Go standard library
- Inspired by similar tools like GoReleaser
