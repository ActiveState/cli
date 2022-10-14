package offinstall

import (
	"path/filepath"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/mitchellh/go-homedir"
)

func DefaultInstallPath() (string, error) {
	home, err := homedir.Dir()
	if err != nil {
		return "", errs.Wrap(err, "Could not get home directory")
	}

	return filepath.Join(home, "Applications"), nil
}
