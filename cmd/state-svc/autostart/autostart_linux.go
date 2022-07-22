package autostart

import "github.com/ActiveState/cli/internal/constants"

func init() {
	Options.LaunchFileName = constants.ServiceLaunchFileName
	Options.IconFileName = constants.ServiceIconFileName
	Options.IconFileSource = constants.IconFileSource
	Options.GenericName = constants.ServiceGenericName
	Options.Comment = constants.ServiceComment
	Options.Keywords = constants.ServiceKeywords
}
