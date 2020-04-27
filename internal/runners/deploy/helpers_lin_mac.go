// +build !windows

package deploy

import (
	"os"
	"path/filepath"

	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/google/uuid"
)

func isWritable(path string) bool {
	// Check if we can write to this path
	fpath := filepath.Join(path, uuid.New().String())
	if err := fileutils.Touch(fpath); err != nil {
		logging.Error("Could not create file: %v", err)
		return false
	}

	if errr := os.Remove(fpath); errr != nil {
		logging.Error("Could not clean up test file: %v", errr)
		return false
	}

	return true
}

func link(src, dst string) error {
	logging.Debug("Creating symlink, oldname: %s newname: %s", src, dst)
	err := os.Symlink(src, dst)
	if err != nil {
		return locale.WrapInputError(
			err, "err_deploy_symlink",
			"Cannot create symlink at {{.V0}}, ensure you have permission to write to {{.V1}}.", dst, filepath.Dir(dst))
	}
	return nil
}

func executable(path string, info os.FileInfo) bool {
	return info.Mode()&0111 != 0
}

func deployMessage() string {
	return locale.T("deploy_restart_shell")
}

func prepareDeployEnv(env map[string]string) {
	return
}
