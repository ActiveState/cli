//go:build darwin
// +build darwin

package installmgr

import (
	"os"
	"path/filepath"

	"github.com/ActiveState/cli/internal/constants"
)

func RemoveSystemFiles(systemInstallPath string) error {
	return os.RemoveAll(filepath.Join(systemInstallPath, constants.MacOSApplicationName))
}
