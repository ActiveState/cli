// +build windows

package deploy

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/go-ole/go-ole"
	"github.com/go-ole/go-ole/oleutil"
)

func isWritable(path string) bool {
	// Avoid writing to paths that require elevated privledges
	avoidPaths := []string{
		"C:\\Windows",
		"C:\\Program Files",
		"C:\\Program Files (x86)",
	}

	info, err := os.Stat(path)
	if err != nil {
		logging.Error("Could not stat path: %s, got error: %v", path, err)
		return false
	}
	if !info.IsDir() {
		return false
	}

	// Check if the user bit is enabled in file permission
	if info.Mode().Perm()&(1<<(uint(7))) == 0 {
		logging.Debug("Write permission bit is not set on: %s", path)
		return false
	}

	for _, a := range avoidPaths {
		if strings.HasPrefix(path, a) {
			return false
		}
	}

	return true
}

func link(src, dst string) error {
	logging.Debug("Creating shortcut, oldname: %s newname: %s", fpath, target)
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

func notExecutable(path string, info os.FileInfo) bool {
	if filepath.Ext(path) == ".exe" {
		return false
	}
	return true
}

func deployMessage() string {
	return locale.T("deploy_restart_cmd")
}

func prepareDeployEnv(env map[string]string) {
	// In order for Windows to find shortcuts on the user PATH
	// we must set the PATHEXT with the correct extension
	originalExtenstions := os.Getenv("PATHEXT")
	env["PATHEXT"] = originalExtenstions + ";.LNK"
}
