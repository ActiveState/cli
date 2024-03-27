package autostart

import (
	svcApp "github.com/ActiveState/cli/cmd/state-svc/app"
	"github.com/ActiveState/cli/internal/config"
	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/logging"
	configMediator "github.com/ActiveState/cli/internal/mediators/config"
	"github.com/ActiveState/cli/internal/multilog"
	"github.com/ActiveState/cli/internal/osutils/autostart"
)

var Options = autostart.Options{
	Name:           constants.SvcAppName,
	LaunchFileName: constants.SvcLaunchFileName,
	Args:           []string{"start", "--autostart"},
}

func RegisterConfigListener(cfg *config.Instance) error {
	app, err := svcApp.New()
	if err != nil {
		return errs.Wrap(err, "Could not init app")
	}

	configMediator.AddListener(constants.AutostartSvcConfigKey, func() {
		if cfg.GetBool(constants.AutostartSvcConfigKey) {
			logging.Debug("Enabling autostart")
			if err := autostart.Enable(app.Path(), Options); err != nil {
				multilog.Error("Failed to enable autostart: %s", errs.JoinMessage(err))
			}
		} else {
			logging.Debug("Disabling autostart")
			if err := autostart.Disable(app.Path(), Options); err != nil {
				multilog.Error("Failed to disable autostart: %s", errs.JoinMessage(err))
			}
		}
	})

	return nil
}
