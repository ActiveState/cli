package prepare

import (
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"runtime"

	"github.com/ActiveState/cli/internal/config"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/osutils"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/primer"
	"github.com/ActiveState/cli/internal/subshell"
	"github.com/ActiveState/cli/internal/subshell/sscommon"
)

type primeable interface {
	primer.Outputer
	primer.Subsheller
}

// Prepare manages the prepare execution context.
type Prepare struct {
	out      output.Outputer
	subshell subshell.SubShell
}

// New prepares a prepare execution context for use.
func New(prime primeable) *Prepare {
	return &Prepare{
		out:      prime.Output(),
		subshell: prime.Subshell(),
	}
}

// Run executes the prepare behavior.
func (r *Prepare) Run() error {
	logging.Debug("ExecutePrepare")

	if runtime.GOOS == "windows" {
		err := setStateProtocol()
		if err != nil {
			r.out.Notice(locale.T("prepare_protocol_warning"))
		}
	}

	binDir := filepath.Join(config.CachePath(), "bin")
	fail := fileutils.Mkdir(binDir)
	if fail != nil {
		return locale.WrapError(fail.ToError(), "err_prepare_bin_dir", "Could not create bin directory")
	}

	envUpdates := map[string]string{
		"PATH": binDir,
	}

	fail = r.subshell.WriteUserEnv(envUpdates, sscommon.Default, true)
	if fail != nil {
		if runtime.GOOS != "linux" {
			return locale.WrapError(fail.ToError(), "err_prepare_update_env", "Could not update user environment")
		}
		logging.Debug("Encountered failure attempting to update user environment: %s", fail.ToError())
		r.out.Notice(locale.T("prepare_env_warning"))
	}

	if runtime.GOOS == "windows" {
		r.out.Print(locale.Tr("prepare_instructions_windows", binDir))
	} else {
		r.out.Print(locale.Tr("prepare_instructions_lin_mac", binDir))
	}

	return nil
}

const (
	protocolKey        = `SOFTWARE\Classes\state`
	protocolCommandKey = `SOFTWARE\Classes\state\shell\open\command`
)

type createKeyFunc = func(path string) (osutils.RegistryKey, bool, error)

func setStateProtocol() error {
	isAdmin, err := osutils.IsWindowsAdmin()
	if err != nil {
		logging.Error("Could not check for windows administrator privileges: %v", err)
	}

	createFunc := osutils.CreateCurrentUserKey
	protocolKeyPath := protocolKey
	protocolCommandKeyPath := protocolCommandKey
	if isAdmin {
		createFunc = osutils.CreateUserKey

		user, err := user.Current()
		if err != nil {
			return locale.WrapError(err, "err_prepare_username", "Could not get current username")
		}
		protocolKeyPath = fmt.Sprintf(`%s\%s`, user.Gid, protocolKey)
		protocolCommandKeyPath = fmt.Sprintf(`%s\%s`, user.Gid, protocolCommandKey)
	}

	protocolKey, _, err := createFunc(protocolKeyPath)
	if err != nil {
		return locale.WrapError(err, "err_prepare_create_protocol_key", "Could not create state protocol registry key")
	}
	defer protocolKey.Close()

	err = protocolKey.SetStringValue("URL Protocol", "")
	if err != nil {
		return locale.WrapError(err, "err_prepare_protocol_set", "Could not set protocol value in registry")
	}

	commandKey, _, err := createFunc(protocolCommandKeyPath)
	if err != nil {
		return locale.WrapError(err, "err_prepare_create_protocol_command_key", "Could not create state protocol command registry key")
	}
	defer commandKey.Close()

	exe, err := os.Executable()
	if err != nil {
		return locale.WrapError(err, "err_prepare_executable", "Could not get current executable")
	}

	err = commandKey.SetStringValue("", fmt.Sprintf(`cmd /k "%s _protocol %%1"`, exe))
	if err != nil {
		return locale.WrapError(err, "err_prepare_command_set", "Could not set command value in registry")
	}

	return nil
}
