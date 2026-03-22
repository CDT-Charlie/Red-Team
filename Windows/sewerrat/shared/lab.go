package shared

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
)

const (
	DefaultMaxDemoCommands = 10
	AuditPreviewLimit      = 160
)

var allowedDemoCommands = map[string]struct{}{
	"arp":        {},
	"dir":        {},
	"echo":       {},
	"getmac":     {},
	"hostname":   {},
	"ipconfig":   {},
	"nslookup":   {},
	"ping":       {},
	"route":      {},
	"set":        {},
	"systeminfo": {},
	"tasklist":   {},
	"type":       {},
	"ver":        {},
	"whoami":     {},
}

var blockedCommandFragments = []string{
	"\n",
	"\r",
	"&&",
	"||",
	"|",
	">",
	"<",
	"`",
}

// LabBaseDir returns the base directory used for lab-only artifacts.
func LabBaseDir() string {
	if configured := strings.TrimSpace(os.Getenv("SEWERRAT_LAB_DIR")); configured != "" {
		return configured
	}

	if runtime.GOOS == "windows" {
		base := strings.TrimSpace(os.Getenv("ProgramData"))
		if base == "" {
			base = os.TempDir()
		}
		return filepath.Join(base, "SewerRatLab")
	}

	return filepath.Join(os.TempDir(), "sewerrat-lab")
}

// AuditDir returns the directory used for audit logs.
func AuditDir() string {
	if configured := strings.TrimSpace(os.Getenv("SEWERRAT_AUDIT_DIR")); configured != "" {
		return configured
	}
	return filepath.Join(LabBaseDir(), "logs")
}

// StopFilePath returns the local kill switch path for the demo implant.
func StopFilePath() string {
	if configured := strings.TrimSpace(os.Getenv("SEWERRAT_STOP_FILE")); configured != "" {
		return configured
	}
	return filepath.Join(LabBaseDir(), "STOP")
}

// SetupAuditLogger configures the standard logger to write to stderr and a lab log file.
func SetupAuditLogger(component string) (string, error) {
	logDir := AuditDir()
	if err := os.MkdirAll(logDir, 0o755); err != nil {
		return "", fmt.Errorf("failed to create audit directory %s: %w", logDir, err)
	}

	logPath := filepath.Join(logDir, component+".log")
	logFile, err := os.OpenFile(logPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return "", fmt.Errorf("failed to open audit log %s: %w", logPath, err)
	}

	log.SetFlags(log.Ldate | log.Ltime | log.Lmicroseconds | log.LUTC)
	log.SetOutput(io.MultiWriter(os.Stderr, logFile))
	return logPath, nil
}

// EnvBool reads a bool-like environment variable with a fallback.
func EnvBool(name string, defaultValue bool) bool {
	switch strings.ToLower(strings.TrimSpace(os.Getenv(name))) {
	case "1", "true", "yes", "on":
		return true
	case "0", "false", "no", "off":
		return false
	default:
		return defaultValue
	}
}

// EnsureDemoMode blocks startup unless the lab-only flag is explicitly set.
func EnsureDemoMode(component string) error {
	if EnvBool("SEWERRAT_DEMO_MODE", false) {
		return nil
	}
	return fmt.Errorf("%s refused to start without SEWERRAT_DEMO_MODE=1", component)
}

// MaxDemoCommands returns the local command budget before the implant exits.
func MaxDemoCommands() int {
	if raw := strings.TrimSpace(os.Getenv("SEWERRAT_MAX_COMMANDS")); raw != "" {
		if value, err := strconv.Atoi(raw); err == nil && value > 0 {
			return value
		}
	}
	return DefaultMaxDemoCommands
}

// KillSwitchPresent returns whether the local stop-file exists.
func KillSwitchPresent() (string, bool, error) {
	path := StopFilePath()
	if _, err := os.Stat(path); err == nil {
		return path, true, nil
	} else if os.IsNotExist(err) {
		return path, false, nil
	} else {
		return path, false, err
	}
}

// SummarizeForAudit compacts free-form text before writing it to logs.
func SummarizeForAudit(value string, limit int) string {
	summary := strings.Join(strings.Fields(strings.TrimSpace(value)), " ")
	if summary == "" {
		return "<empty>"
	}
	if limit > 0 && len(summary) > limit {
		return summary[:limit] + "..."
	}
	return summary
}

// ValidateDemoCommand enforces a read-only command allowlist for the lab implant.
func ValidateDemoCommand(command string) error {
	trimmed := strings.TrimSpace(command)
	if trimmed == "" {
		return fmt.Errorf(ErrEmptyCommand)
	}
	if len(trimmed) > 1024 {
		return fmt.Errorf("command exceeds 1024 characters")
	}

	lower := strings.ToLower(trimmed)
	for _, fragment := range blockedCommandFragments {
		if strings.Contains(lower, fragment) {
			return fmt.Errorf("blocked shell control fragment %q", fragment)
		}
	}

	fields := strings.Fields(lower)
	if len(fields) == 0 {
		return fmt.Errorf(ErrEmptyCommand)
	}

	if _, allowed := allowedDemoCommands[fields[0]]; !allowed {
		return fmt.Errorf("command %q is outside the lab allowlist", fields[0])
	}

	return nil
}
