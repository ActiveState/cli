//go:build windows
// +build windows

package deploy

import (
	"strings"

	"github.com/go-ole/go-ole"
	"github.com/go-ole/go-ole/oleutil"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
)

func shouldSkipSymlink(symlink, fpath string) (bool, error) {
	// If the existing symlink already matches the one we want to create, skip it
	if fileutils.FileExists(symlink) {
		shortcut, err := newShortcut(symlink)
		if err != nil {
			return false, errs.Wrap(err, "Could not create shortcut interface")
		}

		symlinkTarget, err := oleutil.GetProperty(shortcut, "TargetPath")
		if err != nil {
			return false, locale.WrapError(err, "err_link_target", "Could not resolve target of link: {{.V0}}", symlink)
		}

		isAccurate, err := fileutils.PathsEqual(fpath, symlinkTarget.ToString())
		if err != nil {
			return false, locale.WrapError(err, "err_symlink_accuracy_unknown", "Could not determine whether link is owned by State Tool: {{.V0}}.", symlink)
		}
		if isAccurate {
			return true, nil
		}
	}

	return false, nil
}

func link(fpath, symlink string) error {
	if strings.HasSuffix(symlink, ".exe") {
		symlink = strings.Replace(symlink, ".exe", ".lnk", 1)
	}
	logging.Debug("Creating shortcut, destination: %s symlink: %s", fpath, symlink)

	shortcut, err := newShortcut(symlink)
	if err != nil {
		return errs.Wrap(err, "Could not create shortcut interface")
	}

	if _, err = oleutil.PutProperty(shortcut, "TargetPath", fpath); err != nil {
		return errs.Wrap(err, "Could not set TargetPath on lnk")
	}
	if _, err = oleutil.CallMethod(shortcut, "Save"); err != nil {
		return errs.Wrap(err, "Could not save lnk")
	}
	return nil
}

func newShortcut(target string) (*ole.IDispatch, error) {
	// ALWAYS errors with "Incorrect function", which can apparently be safely ignored..
	ole.CoInitialize(0) //nolint:errcheck

	oleShellObject, err := oleutil.CreateObject("WScript.Shell")
	if err != nil {
		return nil, errs.Wrap(err, "Could not create shell object")
	}

	defer oleShellObject.Release()
	wshell, err := oleShellObject.QueryInterface(ole.IID_IDispatch)
	if err != nil {
		return nil, errs.Wrap(err, "Could not interface with shell object")
	}

	defer wshell.Release()
	cs, err := oleutil.CallMethod(wshell, "CreateShortcut", target)
	if err != nil {
		return nil, errs.Wrap(err, "Could not call CreateShortcut on shell object")
	}
	return cs.ToIDispatch(), nil
}
