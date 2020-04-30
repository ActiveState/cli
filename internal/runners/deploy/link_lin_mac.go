// +build !windows

package deploy

import (
	"os"
	"path/filepath"

	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
)

func link(src, dst string) error {
	logging.Debug("Creating symlink, source: %s target: %s", src, dst)
	err := os.Symlink(src, dst)
	if err != nil {
		return locale.WrapInputError(
			err, "err_deploy_symlink",
			"Cannot create symlink at {{.V0}}, ensure you have permission to write to {{.V1}}.", dst, filepath.Dir(dst))
	}
	return nil
}
