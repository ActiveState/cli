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
	logging.Debug("Creating shortcut, oldname: %s newname: %s", src, dst)
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	if strings.HasSuffix(dst, ".exe") {
		dst = strings.Replace(dst, ".exe", ".lnk", 1)
	}

	ole.CoInitializeEx(0, ole.COINIT_APARTMENTTHREADED|ole.COINIT_SPEED_OVER_MEMORY)
	oleShellObject, err := oleutil.CreateObject("WScript.Shell")
	if err != nil {
		return err
	}
	defer oleShellObject.Release()

	wshell, err := oleShellObject.QueryInterface(ole.IID_IDispatch)
	if err != nil {
		return err
	}
	defer wshell.Release()

	cs, err := oleutil.CallMethod(wshell, "CreateShortcut", dst)
	if err != nil {
		return err
	}
	idispatch := cs.ToIDispatch()

	_, err = oleutil.PutProperty(idispatch, "TargetPath", src)
	if err != nil {
		return err
	}

	_, err = oleutil.CallMethod(idispatch, "Save")
	if err != nil {
		return err
	}

	return nil
}

func deployMessage() string {
	return locale.T("deploy_restart_cmd")
}
