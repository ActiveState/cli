// +build !windows

package clean

import (
	"os"
	"path/filepath"
)

func removeConfig(configPath string) error {
	file, err := os.Open(filepath.Join(configPath, "log.txt"))
	if err != nil {
		return err
	}
	err = file.Sync()
	if err != nil {
		return err
	}
	err = file.Close()
	if err != nil {
		return err
	}

	return os.RemoveAll(configPath)
}

func removeInstall(installPath string) error {
	return os.Remove(installPath)
}
