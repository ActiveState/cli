package prepare

import (
	"fmt"
	"os"
	"os/user"
	"path/filepath"

	"github.com/gobuffalo/packr"

	"github.com/ActiveState/cli/internal/appinfo"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/osutils"
	"github.com/ActiveState/cli/internal/osutils/autostart"
	"github.com/ActiveState/cli/internal/osutils/shortcut"
)

func (r *Prepare) prepareOS() error {
	err := setStateProtocol()
	if err != nil {
		r.reportError(locale.T("prepare_protocol_warning"), err)
	}

	trayInfo := appinfo.TrayApp()
	name, exec := trayInfo.Name(), trayInfo.Exec()

	if err := autostart.New(name, exec).Enable(); err != nil {
		r.reportError(locale.Tr("err_prepare_autostart", "Could not enable auto-start, error received: {{.V0}}.", err.Error()), err)
	}

	if err := r.prepareStartShortcut(); err != nil {
		r.reportError(locale.Tr("err_prepare_shortcut", "Could not create start menu shortcut, error received: {{.V0}}.", err.Error()), err)
	}

	return nil
}

func (r *Prepare) prepareStartShortcut() error {
	dir := filepath.Join(os.Getenv("USERPROFILE"), "AppData", "Roaming", "Microsoft", "Windows", "Start Menu", "Programs", "ActiveState")
	if err := fileutils.MkdirUnlessExists(dir); err != nil {
		return locale.WrapInputError(err, "err_preparestart_mkdir", "Could not create start menu entry: %s", dir)
	}

	appInfo := appinfo.TrayApp()
	sc, err := shortcut.New(dir, appInfo.Name(), appInfo.Exec())
	if err != nil {
		return locale.WrapError(err, "err_preparestart_shortcut", "Could not create shortcut")
	}

	box := packr.NewBox("../../../assets")
	if err := sc.SetIconBlob(box.Bytes("icon.ico")); err != nil {
		return locale.WrapError(err, "err_preparestart_icon", "Could not set icon for shortcut file")
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
