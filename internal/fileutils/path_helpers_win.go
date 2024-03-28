//go:build windows
// +build windows

package fileutils

import (
	"syscall"

	"github.com/ActiveState/cli/internal/errs"
)

// GetShortPathName returns the Windows short path (ie., DOS 8.3 notation)
func GetShortPathName(path string) (string, error) {
	p, err := syscall.UTF16FromString(path)
	if err != nil {
		return "", errs.Wrap(err, "failed to convert path to UTF16")
	}
	b := p // GetShortPathName says we can reuse buffer
	n, err := syscall.GetShortPathName(&p[0], &b[0], uint32(len(b)))
	if err != nil {
		return "", err
	}
	if n > uint32(len(b)) {
		b = make([]uint16, n)
		_, err = syscall.GetShortPathName(&p[0], &b[0], uint32(len(b)))
		if err != nil {
			return "", err
		}
	}
	return syscall.UTF16ToString(b), nil
}

// GetLongPathName name returns the Windows long path (ie., DOS 8.3 notation is expanded)
func GetLongPathName(path string) (string, error) {
	p, err := syscall.UTF16FromString(path)
	if err != nil {
		return "", errs.Wrap(err, "failed to convert path to UTF16")
	}
	b := p // GetLongPathName says we can reuse buffer
	n, err := syscall.GetLongPathName(&p[0], &b[0], uint32(len(b)))
	if err != nil {
		return "", err
	}
	if n > uint32(len(b)) {
		b = make([]uint16, n)
		n, err = syscall.GetLongPathName(&p[0], &b[0], uint32(len(b)))
		if err != nil {
			return "", err
		}
	}
	b = b[:n]
	return syscall.UTF16ToString(b), nil
}
