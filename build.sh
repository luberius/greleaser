#!/bin/bash

# build.sh - Script to build optimized binaries for greleaser

# Function to build for a specific platform
build_platform() {
    OS=$1
    ARCH=$2
    OUTPUT="bin/greleaser-${OS}-${ARCH}"
    if [ "$OS" = "windows" ]; then
        OUTPUT="${OUTPUT}.exe"
    fi

    echo "Building for ${OS}/${ARCH}..."
    GOOS=$OS GOARCH=$ARCH go build -ldflags="-s -w" -o $OUTPUT
    
    # Print binary size
    if [ -f "$OUTPUT" ]; then
        size=$(ls -lh "$OUTPUT" | awk '{print $5}')
        echo "Binary size: $size"
        
        # UPX compression if available
        if command -v upx >/dev/null 2>&1; then
            echo "Applying UPX compression..."
            upx --best --lzma "$OUTPUT"
            size=$(ls -lh "$OUTPUT" | awk '{print $5}')
            echo "Compressed size: $size"
        fi
    fi
}

# Create bin directory
mkdir -p bin

# Build for common platforms
build_platform "linux" "amd64"
build_platform "darwin" "amd64"
build_platform "darwin" "arm64"
build_platform "windows" "amd64"
