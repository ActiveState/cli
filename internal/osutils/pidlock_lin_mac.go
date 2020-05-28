// +build linux darwin

package osutils

import (
	"os"
	"syscall"
)

// PidExists checks if a process with the given PID exists and is running
func PidExists(pid int) bool {
	p, err := os.FindProcess(int(pid))
	if err != nil {
		return false
	}
	err = p.Signal(syscall.Signal(0))
	if err == nil {
		return true
	}
	if err.Error() == "os: process already finished" {
		return false
	}
	errno, ok := err.(syscall.Errno)
	if !ok {
		return false
	}
	switch errno {
	case syscall.ESRCH:
		return false
	case syscall.EPERM:
		return true
	}
	return false
}

// LockFile tries to acquire a read lock on the file f
func LockFile(f *os.File) error {
	// attempting to obtain read lock on update file
	ft := &syscall.Flock_t{
		Whence: int16(os.SEEK_SET),
		Start:  0,
		Len:    0,
		Pid:    int32(os.Getpid()),
		Type:   syscall.F_RDLCK,
	}

	return syscall.FcntlFlock(f.Fd(), syscall.F_SETLK, ft)
}

func LockRelease(f *os.File) error {
	ft := &syscall.Flock_t{
		Whence: int16(os.SEEK_SET),
		Start:  0,
		Len:    0,
		Pid:    int32(os.Getpid()),
		Type:   syscall.F_UNLCK,
	}

	return syscall.FcntlFlock(f.Fd(), syscall.F_SETLK, ft)
}

func (pl *PidLock) cleanLockFile(keep bool) error {
	// On Linux we have to remove the file before removing the file lock to avoid race conditions.
	if !keep {
		err := os.Remove(pl.path)
		if err != nil {
			return err
		}
	}
	err := LockRelease(pl.file)
	if err != nil {
		return err
	}
	err = pl.file.Close()
	if err != nil {
		return err
	}
	return nil
}
