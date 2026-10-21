package smartlink

import (
	"os"
)

// linkFile will create a symlink from src to dest, and falls back on a hardlink if no symlink is available.
// This is a workaround for the fact that Windows does not support symlinks without admin privileges.
// src must not be a directory: hard links to directories are not supported.
func linkFile(src, dest string) error {
	return os.Link(src, dest)
}
