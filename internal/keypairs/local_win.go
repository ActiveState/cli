// +build windows

package keypairs

import (
	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/internal/fileutils"
)

func validateKeyFile(keyFilename string) error {
	if !fileutils.FileExists(keyFilename) {
		return FailLoadNotFound.New("keypairs_err_load_not_found")
	}

	return nil
}
