package autostart

import "github.com/ActiveState/cli/internal/constants"

// TODO: Move around/cleanup constants
var data = map[AppName]options{
	Tray: {
		launchFileName: constants.TrayAppName,
		iconFileName:   constants.TrayIconFileName,
		iconFileSource: constants.TrayIconFileSource,
		genericName:    constants.TrayGenericName,
		comment:        constants.TrayComment,
		keywords:       constants.TrayKeywords,
	},
	Service: {
		launchFileName: constants.SvcAppName,
		iconFileName:   constants.TrayIconFileName,
		iconFileSource: constants.TrayIconFileSource,
		genericName:    "Language Runtime Service",
		comment:        "ActiveState Service",
		keywords:       "activestate;state;language;runtime;python;perl;tcl;",
	},
}
