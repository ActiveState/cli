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

	err = LockRead(f)
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

	// write PID in lock file
	_, err = f.Write([]byte(fmt.Sprintf("%d", os.Getpid())))
	if err != nil {
		return nil, err
	}

	// defer release of lock
	return pl, nil
}

// Close removes the lock file and releases the lock
func (pl *PidLock) Close(keepFile ...bool) error {
	keep := false
	if len(keepFile) == 1 {
		keep = keepFile[0]
	}
	err := LockRelease(pl.file)
	if err != nil {
		fmt.Printf("error releasing lock: %v\n", err)
		return err
	}
	err = pl.file.Close()
	if err != nil {
		return err
	}
	if keep {
		return nil
	}
	err = os.Remove(pl.path)
	if err != nil {
		return err
	}
	return nil
}
