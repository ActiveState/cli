// +build darwin

package installation

import (
	"os"
	"path/filepath"

	"github.com/ActiveState/cli/internal/constants"
)

func RemoveSystemFiles(systemInstallPath string) error {
	return os.RemoveAll(filepath.Join(systemInstallPath, constants.MacOSApplicationName))
}
