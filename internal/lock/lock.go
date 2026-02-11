package lock

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"

	"github.com/reloquent/reloquent/internal/config"
)

const DefaultPath = "~/.reloquent/reloquent.lock"

// Acquire creates the lock file with the current process PID.
func Acquire(path string) error {
	if path == "" {
		path = config.ExpandHome(DefaultPath)
	}

	data, err := os.ReadFile(path)
	if err == nil {
		pid, err := strconv.Atoi(strings.TrimSpace(string(data)))
		if err == nil && isProcessRunning(pid) {
			return fmt.Errorf("another Reloquent instance is running (PID %d). Only one migration can run at a time", pid)
		}
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("creating lock directory: %w", err)
	}

	return os.WriteFile(path, []byte(strconv.Itoa(os.Getpid())), 0o644)
}

// Release removes the lock file.
func Release(path string) error {
	if path == "" {
		path = config.ExpandHome(DefaultPath)
	}
	err := os.Remove(path)
	if os.IsNotExist(err) {
		return nil
	}
	return err
}

// IsHeld checks if the lock is currently held by a running process.
func IsHeld(path string) (bool, int, error) {
	if path == "" {
		path = config.ExpandHome(DefaultPath)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return false, 0, nil
		}
		return false, 0, err
	}
	pid, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil {
		return false, 0, nil
	}
	if isProcessRunning(pid) {
		return true, pid, nil
	}
	return false, pid, nil
}

func isProcessRunning(pid int) bool {
	process, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	err = process.Signal(syscall.Signal(0))
	return err == nil
}
