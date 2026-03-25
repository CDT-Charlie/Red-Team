.PHONY: agent admin deps clean help

DIST    := dist
AGENT   := $(DIST)/agent.exe
ADMIN   := $(DIST)/arpshell
LDFLAGS := -s -w

help:
	@echo "Targets:"
	@echo "  agent  - Cross-compile Windows agent (dist/agent.exe)"
	@echo "  admin  - Build Linux admin shell    (dist/arpshell)"
	@echo "  all    - Build both binaries"
	@echo "  deps   - Download and tidy Go modules"
	@echo "  clean  - Remove build artifacts"

deps:
	go mod download
	go mod tidy

agent: deps
	@mkdir -p $(DIST)
	GOOS=windows GOARCH=amd64 go build -trimpath -ldflags "$(LDFLAGS)" -o $(AGENT) ./cmd/agent

admin: deps
	@mkdir -p $(DIST)
	go build -trimpath -ldflags "$(LDFLAGS)" -o $(ADMIN) ./cmd/arpshell

all: agent admin

clean:
	rm -rf $(DIST)
