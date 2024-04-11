package installation

import (
	"path/filepath"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/osutils/user"
)

func installPathForChannel(channel string) (string, error) {
	home, err := user.HomeDir()
	if err != nil {
		return "", errs.Wrap(err, "Could not determine home directory")
	}
	return filepath.Join(home, "AppData", "Local", "ActiveState", "StateTool", channel), nil
}

func defaultSystemInstallPath() (string, error) {
	// There is no system install path for Windows
	return "", nil
}
