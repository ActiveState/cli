package autostart

import (
	"os"
	"path/filepath"

	"github.com/go-ole/go-ole"
	"github.com/go-ole/go-ole/oleutil"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/logging"
)

var startupPath = filepath.Join(os.Getenv("USERPROFILE"), "AppData", "Roaming", "Microsoft", "Windows", "Start Menu", "Programs", "Startup")

func (a *App) Enable() error {
	if a.IsEnabled() {
		return nil
	}

	// ALWAYS errors with "Incorrect function", which can apparently be safely ignored..
	ole.CoInitializeEx(0, ole.COINIT_APARTMENTTHREADED|ole.COINIT_SPEED_OVER_MEMORY)
	defer ole.CoUninitialize()

	oleShellObject, err := oleutil.CreateObject("WScript.Shell")
	if err != nil {
		return errs.Wrap(err, "Could not create shell object")
	}
	defer oleShellObject.Release()

	wshell, err := oleShellObject.QueryInterface(ole.IID_IDispatch)
	if err != nil {
		return errs.Wrap(err, "Could not interface with shell object")
	}
	defer wshell.Release()

	logging.Debug("Creating shortcut: %s", a.shortcutFilename())
	cs, err := oleutil.CallMethod(wshell, "CreateShortcut", a.shortcutFilename())
	if err != nil {
		return errs.Wrap(err, "Could not call CreateShortcut on shell object")
	}
	idispatch := cs.ToIDispatch()

	logging.Debug("Setting TargetPath: %s", a.Exec)
	_, err = oleutil.PutProperty(idispatch, "TargetPath", a.Exec)
	if err != nil {
		return errs.Wrap(err, "Could not set shortcut target")
	}

	_, err = oleutil.CallMethod(idispatch, "Save")
	if err != nil {
		return errs.Wrap(err, "Could not save shortcut")
	}

	return nil
}

func (a *App) Disable() error {
	if !a.IsEnabled() {
		return nil
	}
	return os.Remove(a.shortcutFilename())
}

func (a *App) IsEnabled() bool {
	return fileutils.FileExists(a.shortcutFilename())
}

func (a *App) shortcutFilename() string {
	return filepath.Join(startupPath, a.Name+".lnk")
}
