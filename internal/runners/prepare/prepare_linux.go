package prepare

import (
	"path/filepath"

	svcAutostart "github.com/ActiveState/cli/cmd/state-svc/autostart"
	"github.com/ActiveState/cli/internal/app"
	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/installation"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/osutils/autostart"
	"github.com/ActiveState/cli/internal/osutils/user"
)

func (r *Prepare) prepareOS() error {
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

	if err = autostart.Enable(svcApp.Exec, svcAutostart.Options); err != nil {
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

func cleanOS(cfg app.Configurable) error {
	svcExec, err := installation.ServiceExec()
	if err != nil {
		return locale.WrapError(err, "Could not get state-svc location")
	}
	svcApp, err := app.New(constants.SvcAppName, svcExec, []string{"start"}, app.Options{}, cfg)
	if err != nil {
		return locale.WrapError(err, "Could not get svc autostart shortcut")
	}
	// cleans ~/.profile if necessary
	if err = autostart.Disable(svcApp.Exec, svcAutostart.Options); err != nil {
		return errs.Wrap(err, "Failed to enable autostart for service app.")
	}
}
