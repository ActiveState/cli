// +build darwin

package installation

import (
	"os"
	"path/filepath"
)

const (
	appName = "ActiveState Desktop.app"
)

func RemoveSystemFiles(systemInstallPath string) error {
	return os.RemoveAll(filepath.Join(systemInstallPath, appName))
}
