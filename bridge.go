package main

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"syscall"
	"time"

	"github.com/creack/pty"
)

const (
	agyBinary      = "agy"
	defaultTimeout = 5 * time.Minute
)

// ansiRe strips ANSI escape sequences from terminal output.
var ansiRe = regexp.MustCompile(`\x1b\[[0-9;]*[a-zA-Z]|\x1b\][^\x07]*\x07|\x1b[()][AB012]|\r`)

// RunResult holds the output from a single agy invocation.
type RunResult struct {
	Output string
	Err    error
}

// RunPrompt sends a prompt to agy via pty and returns the full response.
// If sess.HasHistory is true, --continue is passed to resume the last conversation.
func RunPrompt(sess *Session, prompt string) RunResult {
	// First attempt with --print mode
	result := runPromptWithMode(sess, prompt, false)
	
	// If we got empty output and this is a Thinking model, retry with --prompt-interactive
	if result.Output == "" && strings.Contains(sess.Model, "Thinking") {
		result = runPromptWithMode(sess, prompt, true)
	}
	
	return result
}

func runPromptWithMode(sess *Session, prompt string, useInteractive bool) RunResult {
	args := buildArgs(sess, prompt, useInteractive)

	cmd := exec.Command(agyBinary, args...)
	cmd.Dir = sess.Cwd
	cmd.Env = os.Environ()

	// Open a pty to work around agy's TTY detection (Issue #76).
	ptmx, err := pty.Start(cmd)
	if err != nil {
		return RunResult{Err: fmt.Errorf("pty start failed: %w", err)}
	}
	defer ptmx.Close()

	// Set a sane terminal size to prevent layout weirdness.
	pty.Setsize(ptmx, &pty.Winsize{Rows: 50, Cols: 220})

	// Read all output with a global timeout.
	output, err := readWithTimeout(ptmx, defaultTimeout)
	if err != nil && !isPtyEOF(err) {
		cmd.Wait()
		return RunResult{Err: fmt.Errorf("read error: %w", err)}
	}

	cmd.Wait()

	cleaned := cleanOutput(output)
	return RunResult{Output: cleaned}
}

// buildArgs constructs the agy argument list for a prompt invocation.
func buildArgs(sess *Session, prompt string, useInteractive bool) []string {
	if useInteractive {
		// Use --prompt-interactive for models that don't work with --print
		args := []string{"--dangerously-skip-permissions", "--prompt-interactive", prompt}
		if sess.Model != "" {
			args = append([]string{"--model", sess.Model}, args...)
		}
		if sess.HasHistory {
			args = append([]string{"--continue"}, args...)
		}
		return args
	}
	
	args := []string{"--dangerously-skip-permissions", "--print", prompt}
	if sess.Model != "" {
		// Prepend --model before other flags.
		args = append([]string{"--model", sess.Model}, args...)
	}
	if sess.HasHistory {
		// Prepend --continue to resume the last conversation.
		args = append([]string{"--continue"}, args...)
	}
	return args
}

// readWithTimeout reads all bytes from r until EOF or timeout.
func readWithTimeout(r io.Reader, timeout time.Duration) (string, error) {
	done := make(chan struct {
		data string
		err  error
	}, 1)

	go func() {
		var buf bytes.Buffer
		_, err := io.Copy(&buf, r)
		done <- struct {
			data string
			err  error
		}{buf.String(), err}
	}()

	select {
	case result := <-done:
		return result.data, result.err
	case <-time.After(timeout):
		return "", fmt.Errorf("timeout after %s", timeout)
	}
}

// isPtyEOF returns true for errors that indicate normal pty/process termination.
// On Linux, reading from a pty after the child exits yields EIO (syscall.EIO),
// not io.EOF. Both are treated as clean end-of-output.
func isPtyEOF(err error) bool {
	if err == io.EOF {
		return true
	}
	// EIO is returned by the pty master when the child process exits.
	if pathErr, ok := err.(*os.PathError); ok {
		if pathErr.Err == syscall.EIO {
			return true
		}
	}
	return false
}

// cleanOutput strips ANSI codes, carriage returns, and trims whitespace.
func cleanOutput(raw string) string {
	stripped := ansiRe.ReplaceAllString(raw, "")
	// Collapse multiple blank lines to one.
	lines := strings.Split(stripped, "\n")
	var out []string
	blanks := 0
	for _, line := range lines {
		trimmed := strings.TrimRight(line, " \t")
		if trimmed == "" {
			blanks++
			if blanks <= 1 {
				out = append(out, "")
			}
		} else {
			blanks = 0
			out = append(out, trimmed)
		}
	}
	return strings.TrimSpace(strings.Join(out, "\n"))
}
