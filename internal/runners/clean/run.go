package clean

import (
	"os"

	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/runners/prepare"
)

func (u *Uninstall) runUninstall() error {
	// we aggregate installation errors, such that we can display all installation problems in the end
	// TODO: This behavior should be replaced with a proper rollback mechanism https://www.pivotaltracker.com/story/show/178134918
	var aggErr error
	err := removeCache(u.cfg.CachePath())
	if err != nil {
		aggErr = locale.WrapError(aggErr, "uninstall_remove_cache_err", "Failed to remove cache directory {{.V0}}.", u.cfg.CachePath())
	}

	err = removeInstall(u.cfg, u.installDir)
	if err != nil {
		aggErr = locale.WrapError(aggErr, "uninstall_remove_executables_err", "Failed to remove all State Tool files in installation directory {{.V0}}", u.installDir)
	}

	err = removeConfig(u.cfg)
	if err != nil {
		aggErr = locale.WrapError(aggErr, "uninstall_remove_config_err", "Failed to remove configuration directory {{.V0}}", u.cfg.ConfigPath())

	}

	err = undoPrepare()
	if err != nil {
		aggErr = locale.WrapError(aggErr, "uninstall_prepare_err", "Failed to undo some installation steps.")
	}

	if aggErr != nil {
		return aggErr
	}

	u.out.Print(locale.T("clean_success_message"))
	return nil
}

func removeCache(cachePath string) error {
	err := os.RemoveAll(cachePath)
	if err != nil {
		return locale.WrapError(err, "err_remove_cache", "Could not remove State Tool cache directory")
	}
	return nil
}

func undoPrepare() error {
	toRemove := prepare.InstalledPreparedFiles()

	var aggErr error
	for _, f := range toRemove {
		if fileutils.TargetExists(f) {
			err := os.Remove(f)
			if err != nil {
				aggErr = locale.WrapError(aggErr, "err_undo_prepare_remove_file", "Failed to remove file %s", f)
			}
		}
	}

	return aggErr
}
