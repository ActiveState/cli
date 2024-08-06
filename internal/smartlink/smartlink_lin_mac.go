//go:build !windows
// +build !windows

package smartlink

import (
	"os"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/fileutils"
)

// file will create a symlink from src to dest, and falls back on a hardlink if no symlink is available.
// This is a workaround for the fact that Windows does not support symlinks without admin privileges.
func linkFile(src, dest string) error {
	if fileutils.IsDir(src) {
		return errs.New("src is a directory, not a file: %s", src)
	}
	return os.Symlink(src, dest)
}
