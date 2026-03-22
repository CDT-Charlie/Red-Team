package shared

import (
	"encoding/json"
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
	DefaultIntervalSeconds = 30
	DefaultMaxSnapshots    = 0
	AuditPreviewLimit      = 180
)

func envBool(name string) bool {
	switch strings.ToLower(strings.TrimSpace(os.Getenv(name))) {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}

// LabBaseDir returns the folder used for lab-only artifacts.
func LabBaseDir() string {
	if configured := strings.TrimSpace(os.Getenv("RMM_LAB_DIR")); configured != "" {
		return configured
	}

	if runtime.GOOS == "windows" {
		base := strings.TrimSpace(os.Getenv("ProgramData"))
		if base == "" {
			base = os.TempDir()
		}
		return filepath.Join(base, "RMMLab")
	}

	return filepath.Join(os.TempDir(), "rmm-lab")
}

// AuditDir returns the directory used for log files.
func AuditDir() string {
	if configured := strings.TrimSpace(os.Getenv("RMM_AUDIT_DIR")); configured != "" {
		return configured
	}
	return filepath.Join(LabBaseDir(), "logs")
}

// StopFilePath returns the local kill-switch path.
func StopFilePath() string {
	if configured := strings.TrimSpace(os.Getenv("RMM_STOP_FILE")); configured != "" {
		return configured
	}
	return filepath.Join(LabBaseDir(), "STOP")
}

// SetupAuditLogger sends standard logging to stderr and a file-backed audit log.
func SetupAuditLogger(component string) (string, error) {
	logDir := AuditDir()
	if err := os.MkdirAll(logDir, 0o755); err != nil {
		return "", fmt.Errorf("create audit dir %s: %w", logDir, err)
	}

	logPath := filepath.Join(logDir, component+".log")
	logFile, err := os.OpenFile(logPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return "", fmt.Errorf("open audit log %s: %w", logPath, err)
	}

	log.SetFlags(log.Ldate | log.Ltime | log.Lmicroseconds | log.LUTC)
	log.SetOutput(io.MultiWriter(os.Stderr, logFile))
	return logPath, nil
}

// EnsureDemoMode blocks startup unless the lab-only mode is explicitly enabled.
func EnsureDemoMode(component string) error {
	if envBool("RMM_DEMO_MODE") || envBool("RMM_LAB_MODE") {
		return nil
	}
	return fmt.Errorf("%s refused to start without RMM_DEMO_MODE=1", component)
}

// MaxSnapshots returns the local snapshot budget before exit.
func MaxSnapshots() int {
	raw := strings.TrimSpace(os.Getenv("RMM_MAX_SNAPSHOTS"))
	if raw == "" {
		return DefaultMaxSnapshots
	}
	value, err := strconv.Atoi(raw)
	if err != nil || value < 0 {
		return DefaultMaxSnapshots
	}
	return value
}

// SnapshotPath returns the file used for JSON snapshot persistence.
func SnapshotPath() string {
	if configured := strings.TrimSpace(os.Getenv("RMM_SNAPSHOT_PATH")); configured != "" {
		return configured
	}
	return filepath.Join(LabBaseDir(), "snapshots.jsonl")
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

// SummarizeForAudit compacts text before logging it.
func SummarizeForAudit(value string, limit int) string {
	value = strings.Join(strings.Fields(strings.TrimSpace(value)), " ")
	if value == "" {
		return "<empty>"
	}
	if limit > 0 && len(value) > limit {
		return value[:limit] + "..."
	}
	return value
}

// WriteJSONLine writes a single JSON object as a line.
func WriteJSONLine(path string, value any) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}

	file, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer file.Close()

	enc := json.NewEncoder(file)
	return enc.Encode(value)
}
