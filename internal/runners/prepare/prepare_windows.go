package prepare

import (
	"fmt"
	"os"
	"os/user"
	"path/filepath"

	svcApp "github.com/ActiveState/cli/cmd/state-svc/app"
	svcAutostart "github.com/ActiveState/cli/cmd/state-svc/autostart"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/multilog"
	"github.com/ActiveState/cli/internal/osutils"
	"github.com/ActiveState/cli/internal/osutils/autostart"
	"github.com/ActiveState/cli/internal/osutils/shortcut"
)

var shortcutDir = filepath.Join(os.Getenv("USERPROFILE"), "AppData", "Roaming", "Microsoft", "Windows", "Start Menu", "Programs", "ActiveState")

func (r *Prepare) prepareOS() error {
	err := setStateProtocol()
	if err != nil {
		r.reportError(locale.T("prepare_protocol_warning"), err)
	}

	if err := r.prepareStartShortcut(); err != nil {
		r.reportError(locale.Tl("err_prepare_shortcut", "Could not create start menu shortcut. Error received: {{.V0}}.", err.Error()), err)
	}

	a, err := svcApp.New()
	if err != nil {
		return locale.WrapError(err, "err_autostart_app")
	}

	if err = autostart.Enable(a.Path(), svcAutostart.Options); err != nil {
		r.reportError(locale.Tl("err_prepare_service_autostart", "Could not setup service autostart. Error recieved: {{.V0}}", err.Error()), err)
	}

	return nil
}

func (r *Prepare) prepareStartShortcut() error {
	if err := fileutils.MkdirUnlessExists(shortcutDir); err != nil {
		return locale.WrapInputError(err, "err_preparestart_mkdir", "Could not create start menu entry: %s", shortcutDir)
	}

	sc := shortcut.New(shortcutDir, "Uninstall State Tool", r.subshell.Binary(), "/C \"state clean uninstall --prompt\"")
	err := sc.Enable()
	if err != nil {
		return locale.WrapError(err, "err_preparestart_shortcut", "", sc.Path())
	}

	return nil
}

const (
	protocolKey        = `SOFTWARE\Classes\state`
	protocolCommandKey = `SOFTWARE\Classes\state\shell\open\command`
)

func setStateProtocol() error {
	isAdmin, err := osutils.IsAdmin()
	if err != nil {
		multilog.Error("Could not check for windows administrator privileges: %v", err)
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
		protocolKeyPath = fmt.Sprintf(`%s\%s`, user.Uid, protocolKey)
		protocolCommandKeyPath = fmt.Sprintf(`%s\%s`, user.Uid, protocolCommandKey)
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

	err = commandKey.SetStringValue("", fmt.Sprintf(`%s _protocol %%1`, exe))
	if err != nil {
		return locale.WrapError(err, "err_prepare_command_set", "Could not set command value in registry")
	}

	return nil
}

func cleanOS() error {
	return nil
}
