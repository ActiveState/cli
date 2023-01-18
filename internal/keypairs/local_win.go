//go:build windows
// +build windows

package keypairs

import (
	"github.com/ActiveState/cli/internal-as/fileutils"
	"github.com/ActiveState/cli/internal-as/locale"
)

func validateKeyFile(keyFilename string) error {
	if !fileutils.FileExists(keyFilename) {
		return locale.NewError("keypairs_err_load_not_found")
	}

	return nil
}
