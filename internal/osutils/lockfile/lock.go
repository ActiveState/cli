package lockfile

import (
	"fmt"
	"os"
	"time"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/osutils/stacktrace"
)

// Lock represents a lock file that can be used for exclusive access to
// resources that should be accessed by only one process at a time.
//
// The characteristics of the lock are:
// - Lockfiles are NOT removed after use (This seems to create race conditions)  There may be possible solutions out there: https://stackoverflow.com/questions/17708885/flock-removing-locked-file-without-race-condition,
//     but they are not straight-forward, and even in the linked SO question, the answers imply that some of the solutions might leave some loopholes left.
// - Even though the lockfiles are not removed (because a process has been terminated prematurely), it is unlocked
// - On file-systems that support advisory locks via fcntl or LockFileEx, all file system operations are atomic
// - The lock is NOT thread-safe!
//
// Notes:
// - The implementation currently does not support a blocking wait operation that returns once the lock can be acquired. If required, it can be extended this way.
type Lock struct {
	path   string
	file   *os.File
	locked bool
}

// NewLock creates a new Lock that can be used to get exclusive access to resources between processes
func NewLock(path string) (pl *Lock, err error) {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0666)
	if err != nil {
		return nil, errs.Wrap(err, "failed to open lock file %s", path)
	}
	return &Lock{
		path: path,
		file: f,
	}, nil
}

// TryLock attempts to lock the created lock file.
func (pl *Lock) TryLock() (err error) {
	err = LockFile(pl.file)
	if err != nil {
		// if lock cannot be acquired it means that another process is holding the lock
		return NewAlreadyLockedError(err, pl.path, "cannot acquire exclusive lock")
	}

	pl.locked = true
	return nil
}

// Close removes the lock file and releases the lock
func (pl *Lock) Close() error {
	if !pl.locked {
		err := pl.file.Close()
		if err != nil {
			return errs.Wrap(err, "failed to close unlocked lock file %s", pl.path)
		}
		return nil
	}
	err := pl.cleanLockFile()
	if err != nil {
		return errs.Wrap(err, "failed to remove lock file")
	}
	return nil
}

func (pl *Lock) cleanLockFile() error {
	err := LockRelease(pl.file)
	if err != nil {
		return errs.Wrap(err, "failed to release lock on lock file %s", pl.path)
	}
	err = pl.file.Close()
	if err != nil {
		return errs.Wrap(err, "failed to close lock file %s", pl.path)
	}
	return nil
}

// WaitForLock will attempt to acquire the lock for the duration given
func (pl *Lock) WaitForLock(timeout time.Duration) error {
	expiration := time.Now().Add(timeout)
	for {
		err := pl.TryLock()
		if err != nil {
			if !errs.Matches(err, &AlreadyLockedError{}) {
				return errs.Wrap(err, "Could not acquire lock")
			}

			if time.Now().After(expiration) {
				return errs.Wrap(err, "Timed out trying to acquire lock")
			}
			time.Sleep(100 * time.Millisecond)
			continue
		}
		return nil
	}
}

// AlreadyLockedError manages info that clarifies why a lock has failed, but
// is still likely valid.
type AlreadyLockedError struct {
	err   error
	file  string
	msg   string
	stack *stacktrace.Stacktrace
}

// NewAlreadyLockedError returns a new AlreadyLockedError.
func NewAlreadyLockedError(err error, file, msg string) *AlreadyLockedError {
	return &AlreadyLockedError{err, file, msg, stacktrace.Get()}
}

// Error implements the error interface.
func (e *AlreadyLockedError) Error() string {
	return fmt.Sprintf("file %q is already locked: %s", e.file, e.msg)
}

// Unwrap allows the unwrapping of a causing error.
func (e *AlreadyLockedError) Unwrap() error {
	return e.err
}

// Stack implements the errs.WrapperError interface.
func (e *AlreadyLockedError) Stack() *stacktrace.Stacktrace {
	return e.stack
}
