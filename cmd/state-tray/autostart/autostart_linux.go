package autostart

import "github.com/ActiveState/cli/internal-as/constants"

func init() {
	Options.LaunchFileName = constants.TrayLaunchFileName
	Options.IconFileName = constants.TrayIconFileName
	Options.IconFileSource = constants.IconFileSource
	Options.GenericName = constants.TrayGenericName
	Options.Comment = constants.TrayComment
	Options.Keywords = constants.TrayKeywords
}
