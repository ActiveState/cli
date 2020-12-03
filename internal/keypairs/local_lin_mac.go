// +build !windows

package keypairs

import (
	"os"

	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/internal/fileutils"
)

func validateKeyFile(keyFilename string) error {
	if !fileutils.FileExists(keyFilename) {
		return FailLoadNotFound.New("keypairs_err_load_not_found")
	}

	keyFileStat, err := os.Stat(keyFilename)
	if err != nil {
		return FailLoad.Wrap(err)
	}

	// allows u+rw only
	if keyFileStat.Mode()&(0177) > 0 {
		return FailLoadFileTooPermissive.New("keypairs_err_load_requires_mode", keyFilename, "0600")
	}

	return nil
}
