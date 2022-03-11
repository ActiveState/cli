// +build !windows,!darwin

package osutils

import (
	"github.com/ActiveState/cli/internal/logging"
	"github.com/skratchdot/open-golang/open"
)

func OpenURI(input string) error {
	err := open.Run(input)

	if err != nil {
		logging.Warning("open.Run failed, attempting alternative method to open browser. Error received: %v", err)
		err = open.RunWith(input, "x-www-browser")
	}
	return err
}
