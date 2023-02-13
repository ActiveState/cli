// Package osutils exposes some of the osutils internal package.
package osutils

import "github.com/ActiveState/cli/internal-as/osutils"

func IsAdmin() (bool, error) {
	return osutils.IsAdmin()
}
