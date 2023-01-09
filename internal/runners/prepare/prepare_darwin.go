package prepare

import (
	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/installation"
	"github.com/ActiveState/cli/internal/installation/app"
	"github.com/ActiveState/cli/internal/locale"
)

func (r *Prepare) prepareOS() error {
	svcExec, err := installation.ServiceExec()
	if err != nil {
		r.reportError(locale.Tl(
			"err_prepare_service_executable",
			"Could not get service executable: {{.V0}}", err.Error(),
		), err)
	}

	svcShortcut, err := app.New(constants.SvcAppName, svcExec, []string{"start"}, app.Options{}, r.cfg)
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

func installedPreparedFiles(cfg app.Configurable) ([]string, error) {
	return nil, nil
}

func cleanOS(cfg app.Configurable) error {
	return nil
}
