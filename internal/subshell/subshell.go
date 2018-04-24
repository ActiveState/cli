package subshell

import (
	"bytes"
	"os"
	"path/filepath"
	"runtime"
	"sync"

	"github.com/ActiveState/cli/internal/virtualenvironment"
	"github.com/gobuffalo/packr"

	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/pkg/projectfile"
	tempfile "github.com/mash/go-tempfile-suffix"

	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/subshell/bash"
	"github.com/ActiveState/cli/internal/subshell/cmd"
	"github.com/ActiveState/cli/internal/subshell/zsh"
	"github.com/alecthomas/template"
)

// SubShell defines the interface for our virtual environment packages, which should be contained in a sub-directory
// under the same directory as this file
type SubShell interface {
	// Activate the given subshell
	Activate(wg *sync.WaitGroup) error

	// Deactivate the given subshell
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
	var subs SubShell
	switch name {
	case "bash":
		subs = &bash.SubShell{}
	case "zsh":
		subs = &zsh.SubShell{}
	case "cmd.exe":
		subs = &cmd.SubShell{}
	default:
		return nil, failures.FailUser.New(T("error_unsupported_shell", map[string]interface{}{
			"Shell": name,
		}))
	}

	rcFile, err := getRcFile(subs)
	if err != nil {
		return nil, err
	}

	subs.SetBinary(binary)
	subs.SetRcFile(rcFile)
	subs.Activate(wg)

	return subs, err
}

// getRcFile creates a temporary RC file that our shell is initiated from, this allows us to template the logic
// used for initialising the subshell
func getRcFile(v SubShell) (*os.File, error) {
	box := packr.NewBox("../../assets/shells")
	tpl := box.String(v.RcFileTemplate())

	rcData := map[string]interface{}{
		"Project": projectfile.Get(),
		"Env":     virtualenvironment.GetEnv(),
		"WD":      virtualenvironment.WorkingDirectory(),
	}
	t, err := template.New("rcfile").Parse(tpl)
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
