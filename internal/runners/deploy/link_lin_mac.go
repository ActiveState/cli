// +build !windows

package deploy

import (
	"os"
	"path/filepath"

	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
)

func link(fpath, symlink string) error {
	logging.Debug("Creating symlink, destination: %s symlink: %s", fpath, symlink)
	err := os.Symlink(fpath, symlink)
	if err != nil {
		return locale.WrapInputError(
			err, "err_deploy_symlink",
			"Cannot create symlink at {{.V0}}, ensure you have permission to write to {{.V1}}.", symlink, filepath.Dir(symlink))
	}
	return nil
}
