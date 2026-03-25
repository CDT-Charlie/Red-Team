package execution

import (
	"fmt"
	"os/exec"
)

// ExecutePowerShell runs a PowerShell command string on Windows Server 2022.
// It returns the combined stdout/stderr output or an error.
func ExecutePowerShell(command string) (string, error) {
	cmd := exec.Command(
		"powershell.exe",
		"-NoProfile",
		"-NonInteractive",
		"-WindowStyle", "Hidden",
		"-Command", command,
	)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return string(output), fmt.Errorf("powershell: %w: %s", err, string(output))
	}

	return string(output), nil
}
