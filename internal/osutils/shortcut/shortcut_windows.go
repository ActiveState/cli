package shortcut

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/go-ole/go-ole"
	"github.com/go-ole/go-ole/oleutil"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/logging"
)

type Shortcut struct {
	dir      string
	name     string
	target   string
	dispatch *ole.IDispatch
}

func New(dir, name, target string) (*Shortcut, error) {
	// ALWAYS errors with "Incorrect function", which can apparently be safely ignored..
	ole.CoInitializeEx(0, ole.COINIT_APARTMENTTHREADED|ole.COINIT_SPEED_OVER_MEMORY)

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

	filename := filepath.Join(dir, name+".lnk")
	logging.Debug("Creating Shortcut: %s", filename)
	cs, err := oleutil.CallMethod(wshell, "CreateShortcut", filename)
	if err != nil {
		return nil, errs.Wrap(err, "Could not call CreateShortcut on shell object")
	}

	s := &Shortcut{dir, name, target, cs.ToIDispatch()}
	if err := s.setTarget(target); err != nil {
		return nil, errs.Wrap(err, "Could not set Shortcut target")
	}

	return s, nil
}

func (s *Shortcut) setTarget(target string) error {
	logging.Debug("Setting TargetPath: %s", target)
	_, err := oleutil.PutProperty(s.dispatch, "TargetPath", target)
	if err != nil {
		return errs.Wrap(err, "Could not set Shortcut target")
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
