package autostart

import "github.com/ActiveState/cli/internal/constants"

func init() {
	AutostartOptions.LaunchFileName = constants.ServiceLaunchFileName
	AutostartOptions.IconFileName = constants.ServiceIconFileName
	AutostartOptions.IconFileSource = constants.IconFileSource
	AutostartOptions.GenericName = constants.ServiceGenericName
	AutostartOptions.Comment = constants.ServiceComment
	AutostartOptions.Keywords = constants.ServiceKeywords
}
