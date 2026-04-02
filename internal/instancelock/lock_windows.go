//go:build windows

package instancelock

import (
	"errors"
	"os"

	"golang.org/x/sys/windows"
)

const lockRegionSize = 1

func lockFile(file *os.File) error {
	var overlapped windows.Overlapped
	return windows.LockFileEx(
		windows.Handle(file.Fd()),
		windows.LOCKFILE_EXCLUSIVE_LOCK|windows.LOCKFILE_FAIL_IMMEDIATELY,
		0,
		lockRegionSize,
		0,
		&overlapped,
	)
}

func unlockFile(file *os.File) error {
	var overlapped windows.Overlapped
	return windows.UnlockFileEx(
		windows.Handle(file.Fd()),
		0,
		lockRegionSize,
		0,
		&overlapped,
	)
}

func isAlreadyLocked(err error) bool {
	return errors.Is(err, windows.ERROR_LOCK_VIOLATION)
}
