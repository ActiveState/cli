package clean

import (
	"os"

	"github.com/ActiveState/cli/internal/config"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
)

func (u *Uninstall) runUninstall() error {
	err := removeCache(u.cachePath)
	if err != nil {
		return err
	}

	err = removeInstall(u.installPath)
	if err != nil {
		return err
	}

	removeConfig()

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

func removeConfig() {
	logging.Debug("Scheduling the removal of the config directory")
	config.ScheduleRemoval(true)
}
