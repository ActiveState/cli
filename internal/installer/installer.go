package installer

import "github.com/ActiveState/cli/internal/config"

const InstallCfgKey = "installPath"

func InstallPath(cfg *config.Instance) string {
	if path := cfg.GetString(InstallCfgKey); path != "" {
		return path
	}
	return defaultInstallPath()
}
