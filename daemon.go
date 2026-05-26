package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
)

const (
	pidFile = "/tmp/agy-acp-bridge.pid"
	logFile = "/tmp/agy-acp-bridge.log"
)

// startDaemon forks the current binary with the "acp" subcommand in the background.
func startDaemon() {
	if isDaemonRunning() {
		fmt.Println("Daemon is already running")
		return
	}

	execPath, err := resolveExecutable()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error resolving executable: %v\n", err)
		os.Exit(1)
	}

	log, err := os.OpenFile(logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error opening log file: %v\n", err)
		os.Exit(1)
	}
	defer log.Close()

	cmd := exec.Command(execPath, "acp")
	cmd.Stdout = log
	cmd.Stderr = log
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}

	if err := cmd.Start(); err != nil {
		fmt.Fprintf(os.Stderr, "Error starting daemon: %v\n", err)
		os.Exit(1)
	}

	if err := os.WriteFile(pidFile, []byte(fmt.Sprintf("%d", cmd.Process.Pid)), 0644); err != nil {
		fmt.Fprintf(os.Stderr, "Error writing PID file: %v\n", err)
		cmd.Process.Kill()
		os.Exit(1)
	}

	fmt.Printf("Daemon started (PID %d)\n", cmd.Process.Pid)
	fmt.Printf("Logs: %s\n", logFile)
}

// stopDaemon sends SIGTERM to the daemon process and removes the PID file.
func stopDaemon() {
	pidData, err := os.ReadFile(pidFile)
	if err != nil {
		if os.IsNotExist(err) {
			fmt.Println("Daemon is not running")
			return
		}
		fmt.Fprintf(os.Stderr, "Error reading PID file: %v\n", err)
		os.Exit(1)
	}

	var pid int
	fmt.Sscanf(string(pidData), "%d", &pid)

	process, err := os.FindProcess(pid)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error finding process %d: %v\n", pid, err)
		os.Exit(1)
	}

	if err := process.Signal(syscall.SIGTERM); err != nil {
		fmt.Fprintf(os.Stderr, "Error stopping process %d: %v\n", pid, err)
		os.Exit(1)
	}

	os.Remove(pidFile)
	fmt.Printf("Daemon stopped (PID %d)\n", pid)
}

// checkDaemonStatus prints the current daemon status.
func checkDaemonStatus() {
	if isDaemonRunning() {
		pidData, _ := os.ReadFile(pidFile)
		fmt.Printf("Daemon is running (PID %s)\n", string(pidData))
		fmt.Printf("Logs: %s\n", logFile)
	} else {
		fmt.Println("Daemon is not running")
	}
}

// isDaemonRunning returns true if a PID file exists and the process is alive.
func isDaemonRunning() bool {
	pidData, err := os.ReadFile(pidFile)
	if err != nil {
		return false
	}

	var pid int
	fmt.Sscanf(string(pidData), "%d", &pid)

	process, err := os.FindProcess(pid)
	if err != nil {
		return false
	}

	if err := process.Signal(syscall.Signal(0)); err != nil {
		os.Remove(pidFile)
		return false
	}

	return true
}

func resolveExecutable() (string, error) {
	execPath, err := os.Executable()
	if err != nil {
		return "", err
	}
	resolved, err := filepath.EvalSymlinks(execPath)
	if err != nil {
		return execPath, nil
	}
	return resolved, nil
}
