package autostart

import "github.com/ActiveState/cli/internal/constants"

func init() {
	AutostartOptions.MacLabel = "com.activestate." + constants.StateSvcCmd
	AutostartOptions.MacInteractive = false
	AutostartOptions.IconFileSource = "state-tray.icns"
	AutostartOptions.IconFileName = "icon.icns"
}
