package clean

import (
	"errors"
	"os"
	"os/exec"
	"runtime"
	"strings"

	"github.com/ActiveState/cli/internal/config"
	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
)

type confirmAble interface {
	Confirm(message string, defaultChoice bool) (bool, *failures.Failure)
}

type Clean struct {
	confirmer confirmAble
}

type RunParams struct {
	Force bool
}

func NewClean(confirmer confirmAble) *Clean {
	return &Clean{confirmer: confirmer}
}

func (c *Clean) Run(params *RunParams) error {
	return c.run(params)

}

func (c *Clean) run(params *RunParams) error {
	if os.Getenv(constants.ActivatedStateEnvVarName) != "" {
		return errors.New(locale.T("err_clean_activated"))
	}

	if !params.Force {
		ok, fail := c.confirmer.Confirm(locale.T("clean_confirm_remove"), false)
		if fail != nil {
			return fail.ToError()
		}
		if !ok {
			return nil
		}
	}

	configPath := config.ConfigPath()
	cachePath := config.CachePath()

	installPath, err := getInstallPath()
	if err != nil {
		return err
	}

	logging.Debug("Removing config directory: %s", configPath)
	logging.Debug("Removing cache path: %s", cachePath)
	logging.Debug("Removing state tool binary: %s", installPath)

	if file, ok := logging.CurrentHandler().Output().(*os.File); ok {
		file.Sync()
		file.Close()
	}

	err = os.RemoveAll(configPath)
	if err != nil {
		return err
	}

	err = os.RemoveAll(cachePath)
	if err != nil {
		return err
	}

	err = os.Remove(installPath)
	if err != nil {
		return err
	}

	return nil
}

func getInstallPath() (string, error) {
	var finder string
	switch runtime.GOOS {
	case "linux", "darwin":
		finder = "which"
	case "windows":
		finder = "where"
	default:
		return "", errors.New(locale.Tr("err_clean_unsupported_platform", runtime.GOOS))
	}

	cmd := exec.Command(finder, constants.CommandName)
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(string(output)), nil
}
