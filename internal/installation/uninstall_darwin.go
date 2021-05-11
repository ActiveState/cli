// +build darwin

package installation

const (
	appName = "ActiveState Desktop.app"
)

func RemoveSystemFiles(systemInstallPath string) error {
	return os.RemoveAll(filepath.Join(systemInstallPath, appName))
}
