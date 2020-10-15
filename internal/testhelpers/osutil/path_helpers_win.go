// +build windows

package osutil

import "syscall"

// GetLongPathName name returns the Windows long path (ie. ~1 notation is expanded)
func GetLongPathName(path string) (string, error) {
	p := syscall.StringToUTF16(path)
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
