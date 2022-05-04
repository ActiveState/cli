package prepare

import (
	"github.com/ActiveState/cli/internal/installation"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/multilog"
	"github.com/ActiveState/cli/internal/osutils/autostart"
)

func (r *Prepare) prepareOS() {
}

// InstalledPreparedFiles returns the files installed by the prepare command
func InstalledPreparedFiles(cfg autostart.Configurable) ([]string, error) {
	var files []string
	trayInfo, err := installation.NewAppInfo(installation.TrayApp)
	if err != nil {
		return nil, locale.WrapError(err, "err_tray_info")
	}
	name, exec := trayInfo.Name(), trayInfo.Exec()

	sc, err := autostart.New(name, exec, cfg).Path()
	if err != nil {
		multilog.Error("Failed to determine shortcut path for removal: %v", err)
	} else if sc != "" {
		files = append(files, sc)
	}

	return files, nil
}
