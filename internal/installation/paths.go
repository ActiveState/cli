package installation

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/ActiveState/cli/internal/appinfo"
	"github.com/ActiveState/cli/internal/rtutils"
)

func InstallPath() (string, error) {
	// Facilitate use-case of running executables from the build dir while developing
	if !rtutils.BuiltViaCI && strings.Contains(appinfo.StateApp().Exec(), "/build/") {
		return filepath.Dir(appinfo.StateApp().Exec()), nil
	}
	return defaultInstallPath()
}

func LauncherInstallPath() (string, error) {
	if path, ok := os.LookupEnv("_TEST_SYSTEM_PATH"); ok {
		return path, nil
	}
	return defaultSystemInstallPath()
}

func LogfilePath(configPath string, pid int) string {
	return filepath.Join(configPath, fmt.Sprintf("state-installer-%d.log", pid))
}
