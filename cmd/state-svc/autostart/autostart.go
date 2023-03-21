package autostart

import (
	"github.com/ActiveState/cli/internal/config"
	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/installation"
	"github.com/ActiveState/cli/internal/logging"
	configMediator "github.com/ActiveState/cli/internal/mediators/config"
	"github.com/ActiveState/cli/internal/multilog"
	"github.com/ActiveState/cli/internal/osutils/autostart"
)

var Options = autostart.Options{
	Name: constants.SvcAppName,
	Args: []string{"start"},
}

func RegisterConfigListener(cfg *config.Instance) {
	if svcExec, err := installation.ServiceExec(); err == nil {
		configMediator.AddListener(constants.AutostartSvcConfigKey, func() {
			if cfg.GetBool(constants.AutostartSvcConfigKey) {
				logging.Debug("Enabling autostart")
				autostart.Enable(svcExec, Options)
			} else {
				logging.Debug("Disabling autostart")
				autostart.Disable(svcExec, Options)
			}
		})
	} else {
		multilog.Error("Could not add config listener state-svc could not find its executable")
	}
}
