//go:build !windows
// +build !windows

package deploy

import (
	"os"
	"path/filepath"

	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
)

func shouldSkipSymlink(symlink, fpath string) (bool, error) {
	// If the existing symlink already matches the one we want to create, skip it
	if fileutils.IsSymlink(symlink) {
		symlinkTarget, err := fileutils.SymlinkTarget(symlink)
		if err != nil {
			return false, locale.WrapError(err, "err_symlink_target", "Could not resolve target of symlink: {{.V0}}", symlink)
		}

		isAccurate, err := fileutils.PathsEqual(fpath, symlinkTarget)
		if err != nil {
			return false, locale.WrapError(err, "err_symlink_accuracy_unknown", "Could not determine whether symlink is owned by State Tool: {{.V0}}.", symlink)
		}
		if isAccurate {
			return true, nil
		}
	}

	return false, nil
}

func link(fpath, symlink string) error {
	logging.Debug("Creating symlink, destination: %s symlink: %s", fpath, symlink)
	err := os.Symlink(fpath, symlink)
	if err != nil {
		return locale.WrapExternalError(
			err, "err_deploy_symlink",
			"Cannot create symlink at {{.V0}}, ensure you have permission to write to {{.V1}}.", symlink, filepath.Dir(symlink))
	}
	return nil
}
