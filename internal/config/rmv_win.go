// +build windows

package config

import "github.com/ActiveState/cli/internal/embedrun"

func removeConfig(configPath string) error {
	return embedrun.Script("removeConfig", configPath)
}
