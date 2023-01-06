package autostart

import (
	"github.com/ActiveState/cli/internal/config"
	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/installation"
	"github.com/ActiveState/cli/internal/installation/app"
	"github.com/ActiveState/cli/internal/logging"
	configMediator "github.com/ActiveState/cli/internal/mediators/config"
	"github.com/ActiveState/cli/internal/multilog"
	"github.com/ActiveState/cli/internal/osutils/autostart"
)

const (
	App autostart.AppName = constants.SvcAppName
)

var Options = app.Options{}

func RegisterConfigListener(cfg *config.Instance) {
	if svcExec, err := installation.ServiceExec(); err == nil {
		if as, err := app.New(constants.SvcAppName, svcExec, []string{"start"}, Options, cfg); err == nil {
			configMediator.AddListener(constants.AutostartSvcConfigKey, func() {
				if cfg.GetBool(constants.AutostartSvcConfigKey) {
					logging.Debug("Enabling autostart")
					as.EnableAutostart()
				} else {
					logging.Debug("Disabling autostart")
					as.DisableAutostart()
				}
			})
		} else {
			multilog.Error("Could not add config listener: state-svc could not find its autostart")
		}
	} else {
		multilog.Error("Could not add config listener state-svc could not find its executable")
	}
}
