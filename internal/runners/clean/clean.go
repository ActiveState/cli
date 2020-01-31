package clean

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"runtime"

	"github.com/ActiveState/cli/internal/config"
	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/internal/locale"
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
	// TODO:
	// Can't run in activated state
	// Remove language installs
	// Remove config files
	// Remove state tool binary
	// Needs OS Specific implementations to call (Might not be necessary)
	if os.Getenv(constants.ActivatedStateEnvVarName) != "" {
		return errors.New(locale.T("err_clean_activated"))
	}

	ok, fail := c.confirmer.Confirm(locale.T("clean_confirm_remove"), false)
	if fail != nil {
		return fail.ToError()
	}
	if !ok {
		return nil
	}

	switch runtime.GOOS {
	case "linux":
		return runLinux(params)
	case "darwin":
		return runMac(params)
	case "windows":
		return runWindows(params)
	default:
		return errors.New(locale.Tr("err_clean_unsupported_platform", runtime.GOOS))
	}
}

func runLinux(params *RunParams) error {
	return nil
}

func runMac(params *RunParams) error {
	configPath := config.ConfigPath()
	cachePath := config.CachePath()

	cmd := exec.Command("which", "state")
	installPath, err := cmd.Output()
	if err != nil {
		return err
	}

	fmt.Println("Config Path: ", configPath)
	fmt.Println("Cache Path: ", cachePath)
	fmt.Println("Install dir: ", string(installPath))

	err = os.RemoveAll(configPath)
	if err != nil {
		return err
	}

	err = os.RemoveAll(cachePath)
	if err != nil {
		return err
	}

	// It's currently not finding the state tool at this path for some reason
	if _, err := os.Stat(string(installPath)); os.IsNotExist(err) {
		return errors.New("state tool binary does not exist at install path")
	}
	err = os.Remove(string(installPath))
	if err != nil {
		return err
	}

	return nil
}

func runWindows(params *RunParams) error {
	return nil
}
