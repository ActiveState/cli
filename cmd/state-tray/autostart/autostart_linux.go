package autostart

import (
	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/osutils/autostart"
)

var Options = autostart.Options{
	LaunchFileName: constants.TrayAppName,
	IconFileName:   constants.TrayIconFileName,
	IconFileSource: constants.IconFileSource,
	GenericName:    constants.TrayGenericName,
	Comment:        constants.TrayComment,
	Keywords:       constants.TrayKeywords,
	ConfigKey:      "systray.autostarted.disabled",
	SetConfig:      true,
}
