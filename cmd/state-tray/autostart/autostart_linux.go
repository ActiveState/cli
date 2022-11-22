package autostart

import "github.com/ActiveState/cli/internal/constants"

func init() {
	Options.LaunchFileName = constants.TrayLaunchFileName
	Options.IconFileName = constants.TrayIconFileName
	Options.IconFileSource = constants.IconFileSource
	Options.GenericName = constants.TrayGenericName
	Options.Comment = constants.TrayComment
	Options.Keywords = constants.TrayKeywords
}
