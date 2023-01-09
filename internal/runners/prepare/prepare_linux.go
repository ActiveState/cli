package prepare

import (
	"path/filepath"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/installation"
	"github.com/ActiveState/cli/internal/installation/app"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/multilog"
	"github.com/ActiveState/cli/internal/osutils/user"
)

func (r *Prepare) prepareOS() error {
	trayExec, err := installation.TrayExec()
	if err != nil {
		r.reportError(locale.Tl(
			"err_prepare_tray_exec",
			"Could not get tray executable: {{.V0}}", err.Error(),
		), err)
	}

	trayShortcut, err := app.New(constants.TrayAppName, trayExec, nil, app.Options{}, r.cfg)
	if err != nil {
		r.reportError(locale.T("err_autostart_app"), err)
	}

	err = trayShortcut.EnableAutostart()
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

	svcApp, err := app.New(constants.SvcAppName, svcExec, []string{"start"}, app.Options{}, r.cfg)
	if err != nil {
		r.reportError(locale.T("err_autostart_app"), err)
	}

	err = svcApp.EnableAutostart()
	if err != nil {
		r.reportError(locale.Tl(
			"err_prepare_autostart",
			"Could not enable autostart: {{.V0}}.", err.Error(),
		), err)
	}

	return nil
}

func prependHomeDir(path string) (string, error) {
	homeDir, err := user.HomeDir()
	if err != nil {
		return "", errs.Wrap(err, "Could not get home directory")
	}
	return filepath.Join(homeDir, path), nil
}

func installedPreparedFiles(cfg app.Configurable) ([]string, error) {
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

func cleanOS(cfg app.Configurable) error {
	svcExec, err := installation.ServiceExec()
	if err != nil {
		return locale.WrapError(err, "Could not get state-svc location")
	}
	svcApp, err := app.New(constants.SvcAppName, svcExec, []string{"start"}, app.Options{}, cfg)
	if err != nil {
		return locale.WrapError(err, "Could not get svc autostart shortcut")
	}
	return svcApp.DisableAutostart() // cleans ~/.profile if necessary
}
