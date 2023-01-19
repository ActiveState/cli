package legacytray

import (
	"os"
	"path/filepath"

	"github.com/ActiveState/cli/internal-as/errs"
	"github.com/ActiveState/cli/internal-as/fileutils"
	"github.com/ActiveState/cli/internal-as/osutils/shortcut"
)

func osSpecificRemoveTray(installPath, trayExec string) error {
	shortcutDir := filepath.Join(os.Getenv("USERPROFILE"), "AppData", "Roaming", "Microsoft", "Windows", "Start Menu", "Programs", "ActiveState")
	sc := shortcut.New(shortcutDir, trayAppName, trayExec)
	if shortcutFile := filepath.Dir(sc.Path()); fileutils.FileExists(shortcutFile) {
		err := os.Remove(shortcutFile)
		if err != nil {
			errs.Wrap(err, "Unable to remove shortcut file")
		}
	}
	return nil
}
