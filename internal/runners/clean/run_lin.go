//+build linux

package clean

import (
	"os"
	"path/filepath"

	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/mitchellh/go-homedir"
)

const relativeTrayAppPath = "local/share/applications"

func removeTrayApp() error {
	home, err := homedir.Dir()
	if err != nil {
		return locale.WrapError(err, "err_uninstall_home", "Could not get home dir")
	}

	trayAppPath := filepath.Join(home, relativeTrayAppPath)
	if !fileutils.DirExists(trayAppPath) {
		return nil
	}

	err = os.RemoveAll(trayAppPath)
	if err != nil {
		return locale.WrapError(err, "err_remove_tray", "Could not remove state tray")
	}

	return nil
}
