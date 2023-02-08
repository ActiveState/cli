//go:build !windows
// +build !windows

package keypairs

import (
	"os"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/locale"
)

func validateKeyFile(keyFilename string) error {
	if !fileutils.FileExists(keyFilename) {
		return locale.NewInputError("keypairs_err_load_not_found")
	}

	keyFileStat, err := os.Stat(keyFilename)
	if err != nil {
		return errs.Wrap(err, "Could not stat keyFilename: %s", keyFilename)
	}

	// allows u+rw only
	if keyFileStat.Mode()&(0177) > 0 {
		return locale.NewError("keypairs_err_load_requires_mode", "", keyFilename, "0600")
	}

	return nil
}
