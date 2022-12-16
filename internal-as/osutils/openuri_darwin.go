// +build darwin

package osutils

import (
	"github.com/skratchdot/open-golang/open"
)

func OpenURI(input string) error {
	return open.Run(input)
}
