// +build darwin

package deploy

import (
	"os"

	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/thoas/go-funk"
)

func usablePath() (string, error) {
	binDir := "/usr/local/bin"
	if !fileutils.DirExists(binDir) {
		fail := fileutils.Mkdir(binDir)
		if fail != nil {
			return "", locale.WrapError(fail, "deploy_usable_path", "Please ensure '{{.V0}}' exists and is on your PATH.", binDir)
		}
	}

	if !funk.Contains(os.Getenv("PATH"), binDir) {
		return binDir, locale.NewError("err_symlink_bin_macos", "Please ensure '{{.V0}}' exists and is on your PATH.", binDir)
	}

	return binDir, nil
}
