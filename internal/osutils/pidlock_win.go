// +build windows

package osutils

import (
	"fmt"
	"os"
	"syscall"
	"unsafe"
)

var (
	kernel32, _         = syscall.LoadLibrary("kernel32.dll")
	procLockFileEx, _   = syscall.GetProcAddress(kernel32, "LockFileEx")
	procUnlockFileEx, _ = syscall.GetProcAddress(kernel32, "UnlockFileEx")
)

const (
	winLockfileFailImmediately = 0x00000001
	winLockfileExclusiveLock   = 0x00000002
	winLockfileSharedLock      = 0x00000000
)

const ErrorLockViolation syscall.Errno = 0x21 // 33

// Use of 0x00000000 for the shared lock is a guess based on some the MS Windows
// `LockFileEX` docs, which document the `LOCKFILE_EXCLUSIVE_LOCK` flag as:
//
// > The function requests an exclusive lock. Otherwise, it requests a shared
// > lock.
//
// https://msdn.microsoft.com/en-us/library/windows/desktop/aa365203(v=vs.85).aspx

func lockFileEx(handle syscall.Handle, flags uint32, reserved uint32, numberOfBytesToLockLow uint32, numberOfBytesToLockHigh uint32, offset *syscall.Overlapped) (bool, syscall.Errno) {
	r1, _, errNo := syscall.Syscall6(
		uintptr(procLockFileEx),
		6,
		uintptr(handle),
		uintptr(flags),
		uintptr(reserved),
		uintptr(numberOfBytesToLockLow),
		uintptr(numberOfBytesToLockHigh),
		uintptr(unsafe.Pointer(offset)))

	if r1 != 1 {
		if errNo == 0 {
			return false, syscall.EINVAL
		}

		return false, errNo
	}

	return true, 0
}

func unlockFileEx(handle syscall.Handle, reserved uint32, numberOfBytesToLockLow uint32, numberOfBytesToLockHigh uint32, offset *syscall.Overlapped) (bool, syscall.Errno) {
	r1, _, errNo := syscall.Syscall6(
		uintptr(procUnlockFileEx),
		5,
		uintptr(handle),
		uintptr(reserved),
		uintptr(numberOfBytesToLockLow),
		uintptr(numberOfBytesToLockHigh),
		uintptr(unsafe.Pointer(offset)),
		0)

	if r1 != 1 {
		if errNo == 0 {
			return false, syscall.EINVAL
		}

		return false, errNo
	}

	return true, 0
}

// PidExists checks if a process with the given PID exists and is running
func PidExists(pid int) bool {
	_, err := os.FindProcess(int(pid))
	return err == nil
}

func LockRead(f *os.File) error {
	_, errNo := lockFileEx(syscall.Handle(f.Fd()), winLockfileExclusiveLock|winLockfileFailImmediately, 0, 1, 0, &syscall.Overlapped{})

	if errNo > 0 {
		if errNo == ErrorLockViolation || errNo == syscall.ERROR_IO_PENDING {
			return fmt.Errorf("cannot obtain lock")
		}

		return fmt.Errorf("unknown error: %d", errNo)
	}
	return nil
}

func LockRelease(f *os.File) error {
	// mark the file as unlocked
	if _, errNo := unlockFileEx(syscall.Handle(f.Fd()), 0, 1, 0, &syscall.Overlapped{}); errNo > 0 {
		return fmt.Errorf("error releasing windows file lock. errno: %d", errNo)
	}
	return nil
}
