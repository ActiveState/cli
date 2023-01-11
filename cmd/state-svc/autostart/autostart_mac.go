package autostart

import "github.com/ActiveState/cli/internal/constants"

func init() {
	Options.MacLabel = "com.activestate." + constants.StateSvcCmd
	Options.MacInteractive = false
	Options.IconFileSource = "state-tray.icns"
	Options.IconFileName = "icon.icns"
}
