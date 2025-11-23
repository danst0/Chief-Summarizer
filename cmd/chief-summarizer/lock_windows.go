//go:build windows
// +build windows

package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"unsafe"
)

var (
	kernel32         = syscall.NewLazyDLL("kernel32.dll")
	procLockFileEx   = kernel32.NewProc("LockFileEx")
	procUnlockFileEx = kernel32.NewProc("UnlockFileEx")
)

const (
	LOCKFILE_EXCLUSIVE_LOCK   = 0x00000002
	LOCKFILE_FAIL_IMMEDIATELY = 0x00000001
)

func acquireLock() (*os.File, error) {
	// Use system temp directory for lock file
	lockPath := filepath.Join(os.TempDir(), "chief-summarizer.lock")
	
	// Try to open or create the lock file
	lockFile, err := os.OpenFile(lockPath, os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		return nil, fmt.Errorf("failed to create lock file: %w", err)
	}

	// Try to acquire exclusive lock using Windows LockFileEx
	var overlapped syscall.Overlapped
	handle := syscall.Handle(lockFile.Fd())
	
	ret, _, err := procLockFileEx.Call(
		uintptr(handle),
		uintptr(LOCKFILE_EXCLUSIVE_LOCK|LOCKFILE_FAIL_IMMEDIATELY),
		0,
		1, 0, // Lock 1 byte
		uintptr(unsafe.Pointer(&overlapped)),
	)
	
	if ret == 0 {
		lockFile.Close()
		
		// Try to read PID from lock file to show helpful message
		if data, readErr := os.ReadFile(lockPath); readErr == nil && len(data) > 0 {
			if pid, parseErr := strconv.Atoi(strings.TrimSpace(string(data))); parseErr == nil {
				return nil, fmt.Errorf("another instance is already running (PID: %d)", pid)
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
		var overlapped syscall.Overlapped
		handle := syscall.Handle(lockFile.Fd())
		procUnlockFileEx.Call(
			uintptr(handle),
			0,
			1, 0, // Unlock 1 byte
			uintptr(unsafe.Pointer(&overlapped)),
		)
		lockFile.Close()
	}
}
