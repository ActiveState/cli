package prepare

import (
	"path/filepath"

	svcApp "github.com/ActiveState/cli/cmd/state-svc/app"
	svcAutostart "github.com/ActiveState/cli/cmd/state-svc/autostart"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/osutils/autostart"
	"github.com/ActiveState/cli/internal/osutils/user"
)

func (r *Prepare) prepareOS() error {
	svcApp, err := svcApp.New()
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

func cleanOS() error {
	svcApp, err := svcApp.New()
	if err != nil {
		return locale.WrapError(err, "Could not get svc autostart shortcut")
	}
	// cleans ~/.profile if necessary
	if err = autostart.Disable(svcApp.Exec, svcAutostart.Options); err != nil {
		return errs.Wrap(err, "Failed to enable autostart for service app.")
	}

	return nil
}
