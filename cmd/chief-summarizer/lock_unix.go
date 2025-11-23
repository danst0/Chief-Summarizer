//go:build !windows
// +build !windows

package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
)

func acquireLock() (*os.File, error) {
	// Use system temp directory for lock file
	lockPath := filepath.Join(os.TempDir(), "chief-summarizer.lock")
	
	// Try to open or create the lock file
	lockFile, err := os.OpenFile(lockPath, os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		return nil, fmt.Errorf("failed to create lock file: %w", err)
	}

	// Try to acquire exclusive lock (flock)
	err = syscall.Flock(int(lockFile.Fd()), syscall.LOCK_EX|syscall.LOCK_NB)
	if err != nil {
		lockFile.Close()
		
		// Try to read PID from lock file to show helpful message
		if data, readErr := os.ReadFile(lockPath); readErr == nil && len(data) > 0 {
			if pid, parseErr := strconv.Atoi(strings.TrimSpace(string(data))); parseErr == nil {
				// Check if process is still running
				if process, err := os.FindProcess(pid); err == nil {
					if err := process.Signal(syscall.Signal(0)); err == nil {
						return nil, fmt.Errorf("another instance is already running (PID: %d)", pid)
					}
				}
			}
		}
		
		return nil, fmt.Errorf("another instance is already running")
	}

	// Write current PID to lock file
	lockFile.Truncate(0)
	lockFile.Seek(0, 0)
	fmt.Fprintf(lockFile, "%d\n", os.Getpid())
	lockFile.Sync()

	return lockFile, nil
}

func releaseLock(lockFile *os.File) {
	if lockFile != nil {
		syscall.Flock(int(lockFile.Fd()), syscall.LOCK_UN)
		lockFile.Close()
		// Note: We don't delete the lock file to avoid race conditions
	}
}
