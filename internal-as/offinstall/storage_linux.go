package offinstall

import (
	"path/filepath"

	"github.com/ActiveState/cli/internal-as/errs"
	"github.com/mitchellh/go-homedir"
)

func DefaultInstallParentDir() (string, error) {
	home, err := homedir.Dir()
	if err != nil {
		return "", errs.Wrap(err, "Could not get home directory")
	}

	return filepath.Join(home, ".local", "share", "applications"), nil
}
