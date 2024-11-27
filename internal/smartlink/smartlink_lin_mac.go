//go:build !windows
// +build !windows

package smartlink

import (
	"os"
)

// file will create a symlink from src to dest, and falls back on a hardlink if no symlink is available.
// This is a workaround for the fact that Windows does not support symlinks without admin privileges.
func linkFile(src, dest string) error {
	return os.Symlink(src, dest)
}
