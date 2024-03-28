package shortcut

import (
	"errors"
	"os"
	"path/filepath"
	"strings"

	"github.com/ActiveState/cli/internal/locale"
	"github.com/go-ole/go-ole"
	"github.com/go-ole/go-ole/oleutil"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/logging"
)

type WindowStyle int

// Shortcut WindowStyle values
const (
	Normal    WindowStyle = 1
	Maximized             = 3
	Minimized             = 7
)

type Shortcut struct {
	dir      string
	name     string
	target   string
	args     string
	dispatch *ole.IDispatch
}

func New(dir, name, target string, args ...string) *Shortcut {
	return &Shortcut{
		dir, name, target, strings.Join(args, " "), nil,
	}
}

func (s *Shortcut) Enable() error {
	// ALWAYS errors with "Incorrect function", which can apparently be safely ignored..
	_ = ole.CoInitializeEx(0, ole.COINIT_APARTMENTTHREADED|ole.COINIT_SPEED_OVER_MEMORY)

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

	if err := fileutils.MkdirUnlessExists(s.dir); err != nil {
		if os.IsPermission(err) {
			return locale.NewInputError("err_shortcutdir_writable", "", s.dir)
		} else {
			return errs.Wrap(err, "Could not create Shortcut directory")
		}
	}

	filename := filepath.Join(s.dir, s.name+".lnk")
	logging.Debug("Creating Shortcut: %s", filename)
	cs, err := oleutil.CallMethod(wshell, "CreateShortcut", filename)
	if err != nil {
		logging.Debug("OLE Error details: %s", err.Error())
		oleErr := &ole.OleError{}
		if errors.As(err, &oleErr) {
			logging.Debug("OLE Error details: \nCode:%d\nDescription:%s\nError:%s\nString:%s\nSuberror:%s", oleErr.Code(), oleErr.Description(), oleErr.Error(), oleErr.String(), oleErr.SubError().Error())
			return errs.Wrap(err, "oleutil CreateShortcut returned error: %s", oleErr.String())
		}
		return errs.Wrap(err, "Could not call CreateShortcut on shell object")
	}

	s.dispatch = cs.ToIDispatch()

	err = s.setTarget(s.target, s.args)
	if err != nil {
		return errs.Wrap(err, "Could not set Shortcut target")
	}

	return nil
}

func (s *Shortcut) setTarget(target, args string) error {
	logging.Debug("Setting TargetPath: %s", target)
	_, err := oleutil.PutProperty(s.dispatch, "TargetPath", target)
	if err != nil {
		return errs.Wrap(err, "Could not set Shortcut target")
	}

	logging.Debug("Setting Arguments: %s", args)
	_, err = oleutil.PutProperty(s.dispatch, "Arguments", args)
	if err != nil {
		return errs.Wrap(err, "Could not set Shortcut arguments")
	}

	_, err = oleutil.CallMethod(s.dispatch, "Save")
	if err != nil {
		return errs.Wrap(err, "Could not save Shortcut")
	}

	return nil
}

func (s *Shortcut) setIcon(path string) error {
	logging.Debug("Setting Icon: %s", path)
	_, err := oleutil.PutProperty(s.dispatch, "IconLocation", path)
	if err != nil {
		return errs.Wrap(err, "Could not set IconLocation")
	}

	_, err = oleutil.CallMethod(s.dispatch, "Save")
	if err != nil {
		return errs.Wrap(err, "Could not save Shortcut")
	}

	return nil
}

func (s *Shortcut) SetWindowStyle(style WindowStyle) error {
	_, err := oleutil.PutProperty(s.dispatch, "WindowStyle", int(style))
	if err != nil {
		return errs.Wrap(err, "Could not set shortcut to run minimized")
	}

	_, err = oleutil.CallMethod(s.dispatch, "Save")
	if err != nil {
		return errs.Wrap(err, "Could not save Shortcut")
	}

	return nil
}

func (s *Shortcut) SetIconBlob(blob []byte) error {
	logging.Debug("Setting Icon blob")

	filepath := filepath.Join(filepath.Dir(s.target), strings.Split(filepath.Base(s.target), ".")[0]+"_generated.ico")
	if fileutils.FileExists(filepath) {
		if err := os.Remove(filepath); err != nil {
			return errs.Wrap(err, "Could not remove old ico file: %s", filepath)
		}
	}

	err := fileutils.WriteFile(filepath, blob)
	if err != nil {
		return errs.Wrap(err, "Could not create ico file: %s", filepath)
	}

	return s.setIcon(filepath)
}

func (s *Shortcut) Path() string {
	return filepath.Join(s.dir, s.name+".lnk")
}
