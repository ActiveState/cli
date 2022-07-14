//go:build linux
// +build linux

package autostart

import "github.com/ActiveState/cli/internal/constants"

var data = map[AppName]options{
	Tray: {
		launchFileName: constants.TrayAppName,
		iconFileName:   constants.TrayIconFileName,
		iconFileSource: constants.IconFileSource,
		genericName:    constants.TrayGenericName,
		comment:        constants.TrayComment,
		keywords:       constants.TrayKeywords,
	},
	Service: {
		launchFileName: constants.SvcAppName,
		iconFileName:   constants.ServiceIconFileName,
		iconFileSource: constants.IconFileSource,
		genericName:    constants.ServiceGenericName,
		comment:        constants.ServiceComment,
		keywords:       constants.ServiceKeywords,
	},
}
