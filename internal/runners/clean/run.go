package clean

import (
	"os"
	"path/filepath"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/exeutils"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/installation"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/osutils"
	"github.com/ActiveState/cli/internal/runners/prepare"
	"github.com/ActiveState/cli/internal/subshell"
	"github.com/ActiveState/cli/internal/subshell/sscommon"
)

var asFiles = []string{installation.InstallDirMarker, constants.StateInstallerCmd + exeutils.Extension, filepath.Join("system", constants.MacOSApplicationName), "system"}

func removeCache(cachePath string) error {
	err := os.RemoveAll(cachePath)
	if err != nil {
		return locale.WrapError(err, "err_remove_cache", "Could not remove State Tool cache directory")
	}
	return nil
}

func undoPrepare(cfg configurable) error {
	toRemove := prepare.InstalledPreparedFiles(cfg)

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
		return errs.Wrap(err, "Failed to State Tool installation PATH")
	}
	// Default projects will stop working, so we return them from the PATH as well
	if err := s.CleanUserEnv(cfg, sscommon.DefaultID, !isAdmin); err != nil {
		return errs.Wrap(err, "Failed to remove default directory from PATH")
	}

	if err := s.RemoveLegacyInstallPath(cfg); err != nil {
		return errs.Wrap(err, "Failed to remove legacy install path")
	}

	return nil
}

var errDirNotEmpty = errs.New("Not empty")

func removeEmptyDir(dir string) error {
	empty, err := fileutils.IsEmptyDir(dir)
	if err == nil && empty {
		removeErr := os.RemoveAll(dir)
		if err != nil {
			return errs.Wrap(removeErr, "Could not remove directory")
		}
	} else if err != nil {
		return errs.Wrap(err, "Could not check if directory is empty")
	}

	if !empty {
		return errDirNotEmpty
	}

	return nil
}

func cleanInstallDir(dir string) error {
	for _, file := range asFiles {
		f := filepath.Join(dir, file)
		if !fileutils.FileExists(f) {
			continue
		}

		err := os.Remove(f)
		if err != nil {
			return errs.Wrap(err, "Could not remove file: %s", f)
		}
	}

	return nil
}
