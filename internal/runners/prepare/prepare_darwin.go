package prepare

import (
	svcAutostart "github.com/ActiveState/cli/cmd/state-svc/autostart"
	"github.com/ActiveState/cli/internal/app"
	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/installation"
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

	svcShortcut, err := app.New(constants.SvcAppName, svcExec, []string{"start"}, app.Options{})
	if err != nil {
		r.reportError(locale.T("err_autostart_app"), err)
	}

	if err = autostart.Enable(svcShortcut.Exec, svcAutostart.Options); err != nil {
		return errs.Wrap(err, "Failed to enable autostart for service app.")
	}

	return nil
}

func cleanOS() error {
	return nil
}
