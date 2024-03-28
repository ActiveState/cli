package legacytray

import (
	"os"
	"path/filepath"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/osutils/shortcut"
)

const trayLaunchFileName = ""

func osSpecificRemoveTray(installPath, trayExec string) error {
	shortcutDir := filepath.Join(os.Getenv("USERPROFILE"), "AppData", "Roaming", "Microsoft", "Windows", "Start Menu", "Programs", "ActiveState")
	sc := shortcut.New(shortcutDir, trayAppName, trayExec)
	if shortcutFile := filepath.Dir(sc.Path()); fileutils.FileExists(shortcutFile) {
		err := os.Remove(shortcutFile)
		if err != nil {
			return errs.Wrap(err, "Unable to remove shortcut file")
		}
	}
	return nil
}
