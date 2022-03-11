package prepare

import (
	"github.com/ActiveState/cli/internal/appinfo"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/osutils/autostart"
	"github.com/ActiveState/cli/internal/rollbar"
)

func (r *Prepare) prepareOS() {
}

// InstalledPreparedFiles returns the files installed by the prepare command
func InstalledPreparedFiles(cfg autostart.Configurable) []string {
	var files []string
	trayInfo := appinfo.TrayApp()
	name, exec := trayInfo.Name(), trayInfo.Exec()

	sc, err := autostart.New(name, exec, cfg).Path()
	if err != nil {
		logging.Error("Failed to determine shortcut path for removal: %v", err)
		rollbar.Error("Failed to determine shortcut path for removal: %v", err)
	} else if sc != "" {
		files = append(files, sc)
	}

	return files
}
