package startup

import (
	"os"
	"path/filepath"
	"runtime"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/fileutils"
)

func StartStateService() error {
	svcExePath, err := getStateServicePath()
	if err != nil {
		return err
	}

	cmd := newStartServiceCommand(svcExePath)
	if err := cmd.Start(); err != nil {
		return errs.Wrap(err, "Could not start %s", svcExePath)
	}

	return nil
}

func getStateServicePath() (string, error) {
	stateSvcExe := filepath.Join(filepath.Dir(os.Args[0]), "state-svc")
	if runtime.GOOS == "windows" {
		stateSvcExe = stateSvcExe + ".exe"
	}
	if !fileutils.FileExists(stateSvcExe) {
		return "", errs.New("Could not find: %s", stateSvcExe)
	}

	return stateSvcExe, nil
}
