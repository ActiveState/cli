package shortcut

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/ActiveState/cli/internal/assets"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/language"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/scriptfile"
)

type WindowStyle int

// Shortcut WindowStyle values
const (
	Normal    WindowStyle = 1
	Maximized             = 3
	Minimized             = 7
)

type Shortcut struct {
	dir          string
	name         string
	target       string
	args         string
	windowStyle  WindowStyle
	iconLocation string
}

func New(dir, name, target string, args ...string) *Shortcut {
	return &Shortcut{
		dir:    dir,
		name:   name,
		target: target,
		args:   strings.Join(args, " "),
	}
}

func (s *Shortcut) Enable() error {
	scriptName := "createShortcut"
	scriptBlock, err := assets.ReadFileBytes(fmt.Sprintf("scripts/%s.ps1", scriptName))
	if err != nil {
		return errs.Wrap(err, "Could not read script file: %s", scriptName)
	}

	sf, err := scriptfile.New(language.PowerShell, scriptName, string(scriptBlock))
	if err != nil {
		return errs.Wrap(err, "Could not create new scriptfile")
	}

	if !fileutils.DirExists(s.dir) {
		if err := fileutils.Mkdir(s.dir); err != nil {
			return errs.Wrap(err, "Could not create shortcut directory: %s", s.dir)
		}
	}

	args := []string{"-executionpolicy", "bypass", "-File", sf.Filename(), "-dir", s.dir, "-name", s.name, "-target", s.target, "-shortcutArgs", s.args}

	if s.windowStyle != 0 {
		args = append(args, "-windowStyle", fmt.Sprintf("%d", s.windowStyle))
	}

	if s.iconLocation != "" {
		args = append(args, "-iconFile", s.iconLocation)
	}

	cmd := exec.Command("powershell.exe", args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return locale.WrapError(err, "err_clean_start", "Could not create shortcut. Received error: {{.V0}}", string(out))
	}

	return nil
}

func (s *Shortcut) SetWindowStyle(style WindowStyle) {
	s.windowStyle = style
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
	s.iconLocation = filepath

	return nil
}

func (s *Shortcut) Path() string {
	return filepath.Join(s.dir, s.name+".lnk")
}
