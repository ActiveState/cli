package clean

import (
	"os"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/runners/prepare"
)

func (u *Uninstall) runUninstall() error {
	err := removeCache(u.cfg.CachePath())
	if err != nil {
		u.reportError(locale.Tl("uninstall_remove_cache_err", "Failed to remove cache directory."), err)
	}

	err = removeInstall(u.cfg, u.installDir)
	if err != nil {
		u.reportError(locale.Tl("uninstall_remove_executables_err", "Failed to remove all State Tool files in installation directory {{.V0}}", u.installDir), err)
	}

	err = removeConfig(u.cfg)
	if err != nil {
		u.reportError(locale.Tl("uninstall_remove_config_err", "Failed to remove configuration directory {{.V0}}", u.cfg.ConfigPath()), err)
	}

	err = undoPrepare()
	if err != nil {
		u.reportError(locale.Tl("uninstall_prepare_err", "Failed to undo some installation steps."), err)
	}

	u.out.Print(locale.T("clean_success_message"))
	return nil
}

func (u *Uninstall) reportError(msg string, err error) {
	logging.Error("%s: %v", msg, errs.Join(err, ": "))
	u.out.Notice(msg)
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
				aggErr = errs.Wrap(aggErr, "Failed to remove %s: %v", f, err)
			}
		}
	}

	return aggErr
}
