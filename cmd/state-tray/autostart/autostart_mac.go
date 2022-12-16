package autostart

import "github.com/ActiveState/cli/internal-as/constants"

func init() {
	Options.MacLabel = "com.activestate." + constants.StateTrayCmd
	Options.MacInteractive = true
}
