package clean

import (
	"os"

	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/runners/prepare"
)

func (u *Uninstall) runUninstall() error {
	err := removeCache(u.cfg.CachePath())
	if err != nil {
		return err
	}

	err = removeInstall(u.installDir)
	if err != nil {
		return err
	}

	err = removeConfig(u.cfg)
	if err != nil {
		return err
	}

	undoPrepare(u.out)

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

func undoPrepare(out output.Outputer) {
	toRemove := prepare.InstalledPreparedFiles()

	for _, f := range toRemove {
		if fileutils.TargetExists(f) {
			err := os.Remove(f)
			if err != nil {
				out.Notice(locale.Tl("[ERROR]Warning: [/RESET] Could not remove file {{.V0}}.", f))
			}
		}
	}
}
