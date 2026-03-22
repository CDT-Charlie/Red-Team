.PHONY: all agent dev clean dist-clean deps help

DIST_DIR := dist
BUILD_FLAGS := -trimpath
GOOS_WINDOWS := GOOS=windows
GOARCH_AMD64 := GOARCH=amd64

all: deps agent

deps:
	@echo "[*] Checking Go dependencies for RMM..."
	go mod download
	go mod tidy

agent: deps
	@echo "[*] Building Windows RMM agent..."
	@mkdir -p $(DIST_DIR)
	$(GOOS_WINDOWS) $(GOARCH_AMD64) go build $(BUILD_FLAGS) -ldflags "-s -w" \
		-o $(DIST_DIR)/rmm-agent.exe \
		./cmd/rmm-agent
	@echo "[+] Built: $(DIST_DIR)/rmm-agent.exe"

dev: deps
	@echo "[*] Building development agent..."
	@mkdir -p $(DIST_DIR)
	go build -o $(DIST_DIR)/rmm-agent-dev ./cmd/rmm-agent
	@echo "[+] Built development agent"

clean:
	@echo "[*] Cleaning RMM build artifacts..."
	rm -f $(DIST_DIR)/rmm-agent.exe
	rm -f $(DIST_DIR)/rmm-agent-dev

dist-clean: clean
	@echo "[*] Removing dist directory..."
	rm -rf $(DIST_DIR)

help:
	@echo "RMM Blue Team Monitor Build System"
	@echo "=================================="
	@echo ""
	@echo "Targets:"
	@echo "  make all       - Build the agent"
	@echo "  make agent     - Build Windows agent only"
	@echo "  make dev       - Build the agent for the current platform"
	@echo "  make clean     - Remove build artifacts"
	@echo "  make dist-clean - Remove the dist directory"
	@echo "  make deps      - Download and tidy Go dependencies"
	@echo "  make help      - Show this help"
