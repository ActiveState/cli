// +build darwin

package installation

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/ActiveState/cli/internal/constants"
)

func RemoveSystemFiles(systemInstallPath string) error {
	appDir := filepath.Join(systemInstallPath, constants.MacOSApplicationName)
	fmt.Printf("removing %s\n", appDir)
	return os.RemoveAll(appDir)
}
