package prepare

import (
	"github.com/ActiveState/cli/internal/installation"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/multilog"
	"github.com/ActiveState/cli/internal/osutils/autostart"
)

func (r *Prepare) prepareOS() error {
	svcExec, err := installation.ServiceExec()
	if err != nil {
		r.reportError(locale.Tl(
			"err_prepare_service_executable",
			"Could not get service executable: {{.V0}}", err.Error(),
		), err)
	}

	svcShortcut := autostart.New(autostart.Service, svcExec, []string{"start"}, r.cfg)
	err = svcShortcut.Enable()
	if err != nil {
		r.reportError(locale.Tl(
			"err_prepare_autostart",
			"Could not enable autostart: {{.V0}}.", err.Error(),
		), err)
	}

	return nil
}

// InstalledPreparedFiles returns the files installed by the prepare command
func InstalledPreparedFiles(cfg autostart.Configurable) ([]string, error) {
	var files []string
	trayExec, err := installation.TrayExec()
	if err != nil {
		return nil, locale.WrapError(err, "err_tray_exec")
	}

	sc, err := autostart.New(autostart.Tray, trayExec, nil, cfg).Path()
	if err != nil {
		multilog.Error("Failed to determine shortcut path for removal: %v", err)
	} else if sc != "" {
		files = append(files, sc)
	}

	svcExec, err := installation.ServiceExec()
	if err != nil {
		return nil, locale.WrapError(err, "err_svc_exec")
	}

	sc, err = autostart.New(autostart.Service, svcExec, []string{"start"}, cfg).Path()
	if err != nil {
		multilog.Error("Failed to determine shortcut path for removal: %v", err)
	} else if sc != "" {
		files = append(files, sc)
	}

	return files, nil
}
