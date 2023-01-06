package prepare

import (
	svcAutostart "github.com/ActiveState/cli/cmd/state-svc/autostart"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/installation"
	"github.com/ActiveState/cli/internal/installation/app"
	"github.com/ActiveState/cli/internal/locale"
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

	svcShortcut, err := app.New(constants.SvcAppName, svcExec, []string{"start"}, svcAutostart.Options, r.cfg)
	if err != nil {
		r.reportError(locale.T("err_autostart_app"), err)
	}

	err = svcShortcut.EnableAutostart()
	if err != nil {
		r.reportError(locale.Tl(
			"err_prepare_autostart_enable",
			"Could not enable autostart: {{.V0}}.", err.Error(),
		), err)
	}

	return nil
}

func installedPreparedFiles(cfg autostart.Configurable) ([]string, error) {
	return nil, nil
}

func cleanOS(cfg autostart.Configurable) error {
	return nil
}
