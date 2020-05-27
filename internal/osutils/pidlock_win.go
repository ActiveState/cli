// +build windows

package osutils

import (
	"os"
)

// PidExists checks if a process with the given PID exists and is running
func PidExists(pid int) bool {
	p, err := os.FindProcess(int(pid))
	return err == nil
}

func ReadLock(f *os.File) error {
	return nil
}
