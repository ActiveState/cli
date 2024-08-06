package prepare

import (
	svcApp "github.com/ActiveState/cli/cmd/state-svc/app"
	svcAutostart "github.com/ActiveState/cli/cmd/state-svc/autostart"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/osutils/autostart"
)

func (r *Prepare) prepareOS() error {
	svcShortcut, err := svcApp.New()
	if err != nil {
		r.reportError(locale.T("err_autostart_app"), err)
	}

	if err = autostart.Enable(svcShortcut.Path(), svcAutostart.Options); err != nil {
		return errs.Wrap(err, "Failed to enable autostart for service app.")
	}

	return nil
}

func extraInstalledPreparedFiles() []string {
	return nil
}

func cleanOS() error {
	return nil
}
