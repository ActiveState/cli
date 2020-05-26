// +build !windows

package deploy

import (
	"os"
	"path/filepath"

	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
)

func link(dest, name string) error {
	logging.Debug("Creating symlink, destination: %s name: %s", dest, name)
	err := os.Symlink(dest, name)
	if err != nil {
		return locale.WrapInputError(
			err, "err_deploy_symlink",
			"Cannot create symlink at {{.V0}}, ensure you have permission to write to {{.V1}}.", name, filepath.Dir(name))
	}
	return nil
}
