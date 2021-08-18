package shortcut

import (
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/strutils"
)

type SaveOpts struct {
	Name        string
	GenericName string
	Comment     string
	Keywords    string
	IconData    []byte
	IconPath    string
}

func Save(target, path string, opts SaveOpts) (file string, err error) {
	if !fileutils.FileExists(target) {
		return "", errs.New("Target does not exist")
	}

	iconName := filepath.Base(opts.IconPath)
	iconName = strings.TrimSuffix(iconName, filepath.Ext(iconName))

	name := opts.Name
	if name == "" {
		filepath.Base(path)
	}

	data := desktopFileData{
		Name:        name,
		GenericName: opts.GenericName,
		Comment:     opts.Comment,
		Exec:        target,
		Keywords:    opts.Keywords,
		IconName:    iconName,
	}
	desktopFile, err := strutils.ParseTemplate(desktopFileTmpl, data)
	if err != nil {
		return "", errs.Wrap(err, "Could not execute template")
	}

	if err := fileutils.WriteFile(opts.IconPath, opts.IconData); err != nil {
		return "", errs.Wrap(err, "Could not write icon file")
	}

	if err := fileutils.WriteFile(path, []byte(desktopFile)); err != nil {
		return "", errs.Wrap(err, "Could not write desktop file")
	}

	f, err := os.Open(path)
	if err != nil {
		return "", errs.Wrap(err, "Could not open desktop file")
	}
	err = f.Chmod(0770)
	f.Close()
	if err != nil {
		return "", errs.Wrap(err, "Could not make file executable")
	}

	// set the executable as trusted so users do not need to do it manually
	// gio is "Gnome input/output"
	cmd := exec.Command("gio", "set", path, "metadata::trusted", "true")
	stdoutReader, err := cmd.StdoutPipe()
	if err != nil {
		logging.Errorf("Could not obtain stdout pipe from gio cmd: %v", err)
		return path, nil
	}
	stderrReader, err := cmd.StderrPipe()
	if err != nil {
		logging.Errorf("Could not obtain stderr pipe from gio cmd: %v", err)
		return path, nil
	}

	if err = cmd.Start(); err != nil {
		logging.Errorf("Could not start gio cmd: %v", err)
		return path, nil
	}

	stdoutData, err := io.ReadAll(stdoutReader)
	if err != nil {
		logging.Errorf("Could not read stdout pipe of gio cmd: %v", err)
		return path, nil
	}

	stderrData, err := io.ReadAll(stderrReader)
	if err != nil {
		logging.Errorf("Could not read stderr pipe of gio cmd: %v", err)
		return path, nil
	}

	if err = cmd.Wait(); err != nil {
		logging.Errorf(
			"Could not set desktop file as trusted: %v (stdout: %s; stderr: %s)",
			err, stdoutData, stderrData,
		)
	}

	return path, nil
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
