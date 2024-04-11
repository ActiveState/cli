package clean

import (
	"os"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/osutils"
	"github.com/ActiveState/cli/internal/runners/prepare"
	"github.com/ActiveState/cli/internal/subshell"
	"github.com/ActiveState/cli/internal/subshell/sscommon"
)

func removeCache(cachePath string) error {
	err := os.RemoveAll(cachePath)
	if err != nil {
		if osutils.IsAccessDeniedError(err) {
			return locale.WrapInputError(err, "err_remove_cache")
		}
		return locale.WrapError(err, "err_remove_cache")
	}
	return nil
}

func undoPrepare() error {
	err := prepare.CleanOS()
	if err != nil {
		return locale.WrapError(err, "err_prepare_clean", "Could not perform OS-specific cleanup")
	}

	toRemove, err := prepare.InstalledPreparedFiles()
	if err != nil {
		return locale.WrapError(err, "err_prepared_files", "Could not determine files to remove")
	}

	var aggErr error
	for _, f := range toRemove {
		if fileutils.TargetExists(f) {
			err := os.RemoveAll(f)
			if err != nil {
				aggErr = locale.WrapError(aggErr, "err_undo_prepare_remove_file", "Failed to remove file {{.V0}}", f)
			}
		}
	}

	return aggErr
}

func removeEnvPaths(cfg configurable) error {
	isAdmin, err := osutils.IsAdmin()
	if err != nil {
		return errs.Wrap(err, "Could not determine if running as Windows administrator")
	}

	// remove shell file additions
	s := subshell.New(cfg)
	if err := s.CleanUserEnv(cfg, sscommon.InstallID, !isAdmin); err != nil {
		return errs.Wrap(err, "Failed to remove State Tool installation PATH")
	}
	// Default projects will stop working, so we return them from the PATH as well
	if err := s.CleanUserEnv(cfg, sscommon.DefaultID, !isAdmin); err != nil {
		return errs.Wrap(err, "Failed to remove project directory from PATH")
	}

	if err := s.RemoveLegacyInstallPath(cfg); err != nil {
		return errs.Wrap(err, "Failed to remove legacy install path")
	}

	return nil
}
