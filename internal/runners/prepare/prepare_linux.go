package prepare

import (
	"path/filepath"

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

	trayShortcut, err := autostart.New(autostart.Tray, trayExec, nil, r.cfg)
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

	svcShortuct, err := autostart.New(autostart.Service, svcExec, []string{"start"}, r.cfg)
	if err != nil {
		r.reportError(locale.T("err_autostart_app"), err)
	}

	err = svcShortuct.Enable()
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

// InstalledPreparedFiles returns the files installed by state _prepare
func InstalledPreparedFiles(cfg autostart.Configurable) ([]string, error) {
	var files []string
	trayExec, err := installation.TrayExec()
	if err != nil {
		return nil, locale.WrapError(err, "err_tray_exec")
	}

	trayShortcut, err := autostart.New(autostart.Tray, trayExec, nil, cfg)
	if err != nil {
		return nil, locale.WrapError(err, "err_autostart_app")
	}

	path, err := trayShortcut.Path()
	if err != nil {
		multilog.Error("Failed to determine shortcut path for removal: %v", err)
	} else if path != "" {
		files = append(files, path)
	}

	svcExec, err := installation.ServiceExec()
	if err != nil {
		return nil, locale.WrapError(err, "err_svc_exec")
	}

	svcShortuct, err := autostart.New(autostart.Service, svcExec, []string{"start"}, cfg)
	if err != nil {
		return nil, locale.WrapError(err, "err_autostart_app")
	}

	path, err = svcShortuct.Path()
	if err != nil {
		multilog.Error("Failed to determine shortcut path for removal: %v", err)
	} else if path != "" {
		files = append(files, path)
	}

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
