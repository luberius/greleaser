.PHONY: all clean build-all build-local

all: build-all

build-local:
	@echo "Building optimized local binary..."
	@go build -ldflags="-s -w" -o bin/greleaser
	@echo "Done! Binary size:"
	@ls -lh bin/greleaser | awk '{print $$5}'

build-all:
	@chmod +x build.sh
	@./build.sh

clean:
	@echo "Cleaning..."
	@rm -rf bin/
	@echo "Done!"
