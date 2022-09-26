//go:build darwin
// +build darwin

package deploy

import (
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/locale"
)

func usablePath() (string, error) {
	binDir := "/usr/local/bin"
	if !fileutils.DirExists(binDir) {
		err := fileutils.Mkdir(binDir)
		if err != nil {
			return "", locale.WrapError(err, "deploy_usable_path", "Please ensure '{{.V0}}' exists and is on your PATH.", binDir)
		}
	}

	/*if !funk.Contains(os.Getenv("PATH"), binDir) {
		return binDir, locale.NewError("err_symlink_bin_macos", "Please ensure '{{.V0}}' exists and is on your PATH.", binDir)
	}*/

	return binDir, nil
}
