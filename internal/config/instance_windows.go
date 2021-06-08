// +build windows

package config

import (
	"os"
	"path/filepath"
)

func (i *Instance) getLockFile() string {
	if i.lockFile == "" {
		i.lockFile = filepath.Join(i.configDir.Path, "config.lock")
	}

	return i.lockFile
}

// cleanLockFile removes the separate lock file
// The function does not return an error, as there are legitimate cases where it will fail (when another processes has locked the file again)
func (i *Instance) cleanLockFile() {
	os.Remove(i.getLockFile())
}
