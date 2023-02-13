// Package user exposes some of the user internal package.
package user

import (
	"github.com/ActiveState/cli/internal-as/osutils/user"
)

type HomeDirNotFoundError user.HomeDirNotFoundError

func HomeDir() (string, error) {
	return user.HomeDir()
}
