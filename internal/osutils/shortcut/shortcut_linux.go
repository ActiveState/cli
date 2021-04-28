package shortcut

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/strutils"
)

type Shortcut struct {
	target string
	path   string
}

func New(target, path string) (*Shortcut, error) {
	if !fileutils.FileExists(target) {
		return nil, errs.New("Target does not exist")
	}

	scut := Shortcut{
		target: target,
		path:   path,
	}

	return &scut, nil
}

type ShortcutSaveOpts struct {
	GenericName string
	Comment     string
	Keywords    string
	IconData    []byte
	IconPath    string
}

func (s *Shortcut) Save(name string, opts ShortcutSaveOpts) error {
	iconName := filepath.Base(opts.IconPath)
	iconName = strings.TrimSuffix(iconName, filepath.Ext(iconName))

	data := desktopFileData{
		Name:        name,
		GenericName: opts.GenericName,
		Comment:     opts.Comment,
		Exec:        s.target,
		Keywords:    opts.Keywords,
		IconName:    iconName,
	}
	desktopFile, err := strutils.ParseTemplate(desktopFileTmpl, data)
	if err != nil {
		return errs.Wrap(err, "Could not execute template")
	}

	if err := fileutils.WriteFile(opts.IconPath, opts.IconData); err != nil {
		return errs.Wrap(err, "Could not write icon file")
	}

	if err := fileutils.WriteFile(s.path, []byte(desktopFile)); err != nil {
		return errs.Wrap(err, "Could not write desktop file")
	}

	file, err := os.Open(s.path)
	if err != nil {
		return errs.Wrap(err, "Could not open desktop file")
	}
	err = file.Chmod(0770)
	file.Close()
	if err != nil {
		return errs.Wrap(err, "Could not make file executable")
	}

	// set the executable as trusted so users do not need to do it manually
	// gio is "Gnome input/output"
	cmd := exec.Command("gio", "set", s.path, "metadata::trusted", "true")
	if err := cmd.Run(); err != nil {
		logging.Errorf("Could not set desktop file as trusted: %v", err)
	}

	return nil
}

type desktopFileData struct {
	Name        string
	GenericName string
	Comment     string
	Exec        string
	Keywords    string
	IconName    string
}

var desktopFileTmpl = strings.TrimPrefix(`
[Desktop Entry]
Name={{ .Name }}
GenericName={{ .GenericName }}
Type=Application
Comment={{ .Comment }}
Exec="{{ .Exec }}"
Terminal=false
Keywords={{ .Keywords }}
Categories=Utility;Development;
Hidden=false
NoDisplay=false
StartupNotify=false
Icon={{ .IconName }}
Name[en_US]={{ .Name }}
`, "\n")
