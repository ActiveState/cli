package subshell

import (
	"bytes"
	"os"
	"path/filepath"
	"runtime"
	"sync"

	"github.com/ActiveState/ActiveState-CLI/internal/failures"
	"github.com/ActiveState/ActiveState-CLI/internal/files"
	"github.com/ActiveState/ActiveState-CLI/internal/logging"
	"github.com/ActiveState/ActiveState-CLI/pkg/projectfile"
	tempfile "github.com/mash/go-tempfile-suffix"

	"github.com/ActiveState/ActiveState-CLI/internal/locale"
	"github.com/ActiveState/ActiveState-CLI/internal/subshell/bash"
	"github.com/ActiveState/ActiveState-CLI/internal/subshell/cmd"
	"github.com/alecthomas/template"
)

// SubShell defines the interface for our virtual environment packages, which should be contained in a sub-directory
// under the same directory as this file
type SubShell interface {
	// Activate the given subshell venv
	Activate(wg *sync.WaitGroup) error

	// Deactivate the given subshell venv
	Deactivate() error

	// IsActive returns whether the given subshell is active
	IsActive() bool

	// Binary returns the configured binary
	Binary() string

	// SetBinary sets the configured binary, this should only be called by the subshell package
	SetBinary(string)

	// RcFile returns the parsed RcFileTemplate file to initialise the shell
	RcFile() *os.File

	// SetRcFile sets the configured RC file, this should only be called by the subshell package
	SetRcFile(*os.File)

	// RcFileTemplate returns the file name of the projects terminal config script used to generate project specific terminal configuration scripts, this script should live under assets/shells
	RcFileTemplate() string

	// RcFileExt returns the extension to use (including the dot), primarily aimed at windows
	RcFileExt() string

	// Shell returns an identifiable string representing the shell, eg. bash, zsh
	Shell() string
}

// Activate the virtual environment
func Activate(wg *sync.WaitGroup) (SubShell, error) {
	logging.Debug("Activating Subshell")

	var T = locale.T
	var binary string
	if runtime.GOOS == "windows" {
		binary = os.Getenv("ComSpec")
	} else {
		binary = os.Getenv("SHELL")
	}

	name := filepath.Base(binary)

	var err error
	var venv SubShell
	switch name {
	case "bash":
		venv = &bash.SubShell{}
	case "cmd.exe":
		venv = &cmd.SubShell{}
	default:
		return nil, failures.User.New(T("error_unsupported_shell", map[string]interface{}{
			"Shell": name,
		}))
	}

	rcFile, err := getRcFile(venv)
	if err != nil {
		return nil, err
	}

	venv.SetBinary(binary)
	venv.SetRcFile(rcFile)
	venv.Activate(wg)

	return venv, err
}

// getRcFile creates a temporary RC file that our shell is initiated from, this allows us to template the logic
// used for initialising the subshell
func getRcFile(v SubShell) (*os.File, error) {
	tplFile, err := files.AssetFS.Asset(filepath.Join("shells", v.RcFileTemplate()))
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

	tmpFile, err := tempfile.TempFileWithSuffix(os.TempDir(), "state-subshell-rc", v.RcFileExt())

	if err != nil {
		return nil, err
	}

	tmpFile.WriteString(out.String())
	tmpFile.Close()

	return tmpFile, err
}
