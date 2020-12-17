// +build !windows

package config

import "os"

func removeConfig(configPath string) error {
	return os.RemoveAll(configPath)
}
