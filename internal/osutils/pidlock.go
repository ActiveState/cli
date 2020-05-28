package osutils

import (
	"fmt"
	"io"
	"os"

	"strconv"

	"github.com/ActiveState/cli/internal/errs"
)

// PidLock represents a lock file that can be used for exclusive access to
// resources that should be accessed by only one process at a time.
//
// The characteristics of the lock are:
// - Lockfiles are removed after use
// - Even if the lockfiles are not removed (because a process has been terminated prematurely), it is unlocked
// - On file-systems that support advisory locks via fcntl or LockFileEx, all file system operations are atomic
//
// Notes:
// - The implementation currently does not support a blocking wait operation that returns once the lock can be acquired. If required, it can be extended this way.
// - Storing the PID inside the lockfile was initially intended to be fall-back mechanism for file systems that do not support locking files.  This is probably unnecessary, but could be extended to communicate with the process currently holding the lock via its PID.
type PidLock struct {
	path string
	file *os.File
}

// NewPidLock creates a new PidLock that can be used to get exclusive access to resources between processes
// If the file at path has been created by a different process and that process is still running, the function returns nil and an error.
func NewPidLock(path string) (pl *PidLock, err error) {
	pl = &PidLock{
		path: path,
	}

	f, err := os.OpenFile(pl.path, os.O_CREATE|os.O_RDWR, 0666)
	if err != nil {
		return nil, err
	}
	pl.file = f

	err = LockFile(f)
	if err != nil {
		// if lock cannot be acquired it usually means that another process is holding the lock
		f.Close()
		return nil, err
	}

	// check if PID can be read and if so, if the process is running
	b := make([]byte, 100)
	n, err := f.Read(b)
	if err != nil && err != io.EOF {
		f.Close()
		return nil, err
	}
	if n > 0 {
		pid, err := strconv.ParseInt(string(b[:n]), 10, 64)
		if err != nil {
			f.Close()
			return nil, err
		}
		if PidExists(int(pid)) {
			f.Close()
			return nil, errs.New("cannot acquire lock: pid %d exists", pid)
		}
	}

	// write PID into lock file
	_, err = f.Write([]byte(fmt.Sprintf("%d", os.Getpid())))
	if err != nil {
		return nil, err
	}

	return pl, nil
}

// Close removes the lock file and releases the lock
func (pl *PidLock) Close(keepFile ...bool) error {
	keep := false
	if len(keepFile) == 1 {
		keep = keepFile[0]
	}
	err := pl.cleanLockFile(keep)
	if err != nil {
		return err
	}
	return nil
}
