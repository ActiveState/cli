// +build windows

package deploy

import (
	"runtime"
	"strings"

	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/go-ole/go-ole"
	"github.com/go-ole/go-ole/oleutil"
)

func link(src, dst string) error {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	if strings.HasSuffix(dst, ".exe") {
		dst = strings.Replace(dst, ".exe", ".lnk", 1)
	}
	logging.Debug("Creating shortcut, oldname: %s newname: %s", src, dst)

	ole.CoInitializeEx(0, ole.COINIT_APARTMENTTHREADED|ole.COINIT_SPEED_OVER_MEMORY)
	oleShellObject, err := oleutil.CreateObject("WScript.Shell")
	if err != nil {
		return locale.WrapInputError(
			err, "err_create_shell",
			"Could not create OLE shell object")
	}
	defer oleShellObject.Release()

	wshell, err := oleShellObject.QueryInterface(ole.IID_IDispatch)
	if err != nil {
		return locale.WrapInputError(
			err, "err_create_wshell",
			"Could not create WShell dispatch")
	}
	defer wshell.Release()

	cs, err := oleutil.CallMethod(wshell, "CreateShortcut", dst)
	if err != nil {
		return locale.WrapInputError(
			err, "err_create_shortcut",
			"Could not create shortcut at path: {{.V0}}", dst)
	}
	idispatch := cs.ToIDispatch()

	_, err = oleutil.PutProperty(idispatch, "TargetPath", src)
	if err != nil {
		return locale.WrapInputError(
			err, "err_short_add_target",
			"Could not add target property to shortcut")
	}

	_, err = oleutil.CallMethod(idispatch, "Save")
	if err != nil {
		return locale.WrapInputError(
			err, "err_save_shortcut",
			"Could not save shortcut")
	}

	return nil
}
