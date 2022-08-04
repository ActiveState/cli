package prepare

import (
	"path/filepath"

	svcAutostart "github.com/ActiveState/cli/cmd/state-svc/autostart"
	trayAutostart "github.com/ActiveState/cli/cmd/state-tray/autostart"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/installation"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/multilog"
	"github.com/ActiveState/cli/internal/osutils/autostart"
	"github.com/mitchellh/go-homedir"
)

func (r *Prepare) prepareOS() error {
	trayExec, err := installation.TrayExec()
	if err != nil {
		r.reportError(locale.Tl(
			"err_prepare_tray_exec",
			"Could not get tray executable: {{.V0}}", err.Error(),
		), err)
	}

	trayShortcut, err := autostart.New(trayAutostart.App, trayExec, nil, trayAutostart.Options, r.cfg)
	if err != nil {
		r.reportError(locale.T("err_autostart_app"), err)
	}

	err = trayShortcut.Enable()
	if err != nil {
		r.reportError(locale.Tl(
			"err_prepare_autostart_enable",
			"Could not enable autostart: {{.V0}}.", err.Error(),
		), err)
	}

	svcExec, err := installation.ServiceExec()
	if err != nil {
		r.reportError(locale.Tl(
			"err_prepare_service_executable",
			"Could not get service executable: {{.V0}}", err.Error(),
		), err)
	}

	svcShortcut, err := autostart.New(svcAutostart.App, svcExec, []string{"start"}, svcAutostart.Options, r.cfg)
	if err != nil {
		r.reportError(locale.T("err_autostart_app"), err)
	}

	err = svcShortcut.Enable()
	if err != nil {
		r.reportError(locale.Tl(
			"err_prepare_autostart",
			"Could not enable autostart: {{.V0}}.", err.Error(),
		), err)
	}

	return nil
}

func prependHomeDir(path string) (string, error) {
	homeDir, err := homedir.Dir()
	if err != nil {
		return "", errs.Wrap(err, "Could not get home directory")
	}
	return filepath.Join(homeDir, path), nil
}

func installedPreparedFiles(cfg autostart.Configurable) ([]string, error) {
	var files []string
	dir, err := prependHomeDir(constants.ApplicationDir)
	if err != nil {
		multilog.Error("Failed to set application dir: %v", err)
	} else {
		files = append(files, filepath.Join(dir, constants.TrayLaunchFileName))
	}

	iconsDir, err := prependHomeDir(constants.IconsDir)
	if err != nil {
		multilog.Error("Could not find icons directory: %v", err)
	} else {
		files = append(files, filepath.Join(iconsDir, constants.TrayIconFileName))
	}

	return files, nil
}
