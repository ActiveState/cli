// +build darwin

package deploy

import (
	"os"

	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/output"
	"github.com/thoas/go-funk"
)

func usablePath(out output.Outputer) (string, error) {
	binDir := "/usr/local/bin"
	if !fileutils.DirExists(binDir) {
		fail := fileutils.Mkdir(binDir)
		if fail != nil {
			return "", locale.WrapError(fail.ToError(), "err_symlink_bin_macos", "Could not create {{.V0}} directory for symlinking", binDir)
		}
	}

	if !funk.Contains(os.Getenv("PATH"), binDir) {
		out.Notice(locale.Tr("deploy_usable_path", binDir))
	}

	return binDir, nil
}
