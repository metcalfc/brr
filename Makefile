# Brr - Terminal Speed Reading Tool
# Build both CLI (brr) and GUI (grr) applications

.PHONY: all build brr grr test clean install uninstall fmt lint help

# Default target
all: build

# Build both applications
build: brr grr

# Build the CLI application
brr:
	go build -o brr .

# Build the GUI application (requires Fyne)
grr:
	go build -tags gui -o grr .

# Run tests
test:
	go test -v ./...

# Run benchmarks
bench:
	go test -bench=. ./...

# Clean build artifacts
clean:
	rm -f brr grr
	go clean

# Install binaries to GOPATH/bin
install: build
	go install .
	go install -tags gui .

# Uninstall binaries from GOPATH/bin
uninstall:
	rm -f $(shell go env GOPATH)/bin/brr
	rm -f $(shell go env GOPATH)/bin/grr

# Format code
fmt:
	go fmt ./...

# Run linter (requires golangci-lint)
lint:
	@which golangci-lint > /dev/null || (echo "golangci-lint not found, skipping" && exit 0)
	golangci-lint run

# Tidy dependencies
tidy:
	go mod tidy

# Verify dependencies
verify:
	go mod verify

# Run the CLI with sample file
run: brr
	./brr sample.txt

# Run the GUI with sample file
run-gui: grr
	./grr sample.txt

# Show help
help:
	@echo "Brr - Terminal Speed Reading Tool"
	@echo ""
	@echo "Targets:"
	@echo "  all       Build both brr (CLI) and grr (GUI)"
	@echo "  build     Same as 'all'"
	@echo "  brr       Build only the CLI application"
	@echo "  grr       Build only the GUI application"
	@echo "  test      Run all tests"
	@echo "  bench     Run benchmarks"
	@echo "  clean     Remove build artifacts"
	@echo "  install   Install binaries to GOPATH/bin"
	@echo "  uninstall Remove binaries from GOPATH/bin"
	@echo "  fmt       Format Go source files"
	@echo "  lint      Run golangci-lint (if installed)"
	@echo "  tidy      Tidy go.mod dependencies"
	@echo "  verify    Verify dependencies"
	@echo "  run       Build and run CLI with sample.txt"
	@echo "  run-gui   Build and run GUI with sample.txt"
	@echo "  help      Show this help message"
