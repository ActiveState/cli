package virtualenvironment

import (
	"bytes"
	"errors"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"

	"github.com/ActiveState/ActiveState-CLI/internal/environment"

	"github.com/ActiveState/ActiveState-CLI/pkg/projectfile"
	"github.com/dvirsky/go-pylog/logging"

	"github.com/ActiveState/ActiveState-CLI/internal/locale"
	"github.com/ActiveState/ActiveState-CLI/internal/virtualenvironment/bash"
	"github.com/alecthomas/template"
	"github.com/gchaincl/gotic/fs"
)

// VirtualEnvironment defines the interface for our virtual environment packages, which should be contained in a sub-directory
// under the same directory as this file
type VirtualEnvironment interface {
	// Activate the given subshell venv
	Activate() error

	// Deactivate the given subshell venv
	Deactivate() error

	// IsActive returns whether the given subshell is active
	IsActive() bool

	// GetBinary returns the configured binary
	GetBinary() string

	// SetBinary sets the configured binary, this should only be called by the virtualenvironment package
	SetBinary(string)

	// GetRcFile returns the configured RC file
	GetRcFile() *os.File

	// SetRcFile sets the configured RC file, this should only be called by the virtualenvironment package
	SetRcFile(os.File)

	// Shell returns an identifiable string representing the shell, eg. bash, zsh
	Shell() string

	// ShellScript returns the file name for the rc script used to initialise the shell, this script should live under assets/shells
	ShellScript() string
}

// Activate the virtual environment
func Activate() (VirtualEnvironment, error) {
	logging.Debug("Activating Virtual Environment")

	var T = locale.T

	binary := os.Getenv("SHELL")
	name := path.Base(binary)

	var venv VirtualEnvironment
	var err error

	switch name {
	case "bash":
		venv = &bash.VirtualEnvironment{}
	default:
		return nil, errors.New(T("error_unsupported_shell", map[string]interface{}{
			"Shell": name,
		}))
	}

	rcFile, err := getRcFile(venv)
	if err != nil {
		return nil, err
	}

	venv.SetBinary(binary)
	venv.SetRcFile(*rcFile)
	venv.Activate()

	return venv, err
}

// getRcFile creates a temporary RC file that our shell is initiated from, this allows us to template the logic
// used for initialising the subshell
func getRcFile(v VirtualEnvironment) (*os.File, error) {
	root, err := environment.GetRootPath()
	if err != nil {
		return nil, err
	}

	tplFile, err := fs.ReadFile(filepath.Join(root, "assets", "shells", v.ShellScript()))
	if err != nil {
		return nil, err
	}

	rcData, err := projectfile.Get()
	if err != nil {
		return nil, err
	}

	t, err := template.New("rcfile").Parse(string(tplFile))
	if err != nil {
		return nil, err
	}

	var out bytes.Buffer
	err = t.Execute(&out, rcData)

	tmpFile, err := ioutil.TempFile(os.TempDir(), "state-subshell-rc")

	if err != nil {
		return nil, err
	}

	tmpFile.WriteString(out.String())
	tmpFile.Close()

	return tmpFile, err
}
