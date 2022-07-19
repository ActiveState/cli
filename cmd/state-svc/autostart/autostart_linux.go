package autostart

import (
	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/osutils/autostart"
)

var Options = autostart.Options{
	LaunchFileName: constants.SvcAppName,
	IconFileName:   constants.ServiceIconFileName,
	IconFileSource: constants.IconFileSource,
	GenericName:    constants.ServiceGenericName,
	Comment:        constants.ServiceComment,
	Keywords:       constants.ServiceKeywords,
}
