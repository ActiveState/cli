package subshell

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"

	"github.com/ActiveState/cli/internal/print"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/subshell/bash"
	"github.com/ActiveState/cli/internal/subshell/cmd"
	"github.com/ActiveState/cli/internal/subshell/fish"
	"github.com/ActiveState/cli/internal/subshell/tcsh"
	"github.com/ActiveState/cli/internal/subshell/zsh"
	"github.com/ActiveState/cli/internal/virtualenvironment"
	"github.com/ActiveState/cli/pkg/project"
	"github.com/ActiveState/cli/pkg/projectfile"
	"github.com/alecthomas/template"
	"github.com/gobuffalo/packr"
	tempfile "github.com/mash/go-tempfile-suffix"
	ps "github.com/mitchellh/go-ps"
)

// SubShell defines the interface for our virtual environment packages, which should be contained in a sub-directory
// under the same directory as this file
type SubShell interface {
	// Activate the given subshell
	Activate(wg *sync.WaitGroup) error

	// Deactivate the given subshell
	Deactivate() error

	// Run a command string that assumes this shell
	Run(script string) error

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

	// Why another check here? Because some things like hooks / run command don't take the virtualenv route,
	// realistically this shouldn't really happen, but it's a useful failsafe for us
	activeProject := os.Getenv(constants.ActivatedStateEnvVarName)
	if activeProject != "" {
		return nil, virtualenvironment.FailAlreadyActive.New("err_already_active")
	}

	subs, fail := Get()
	if fail != nil {
		return nil, fail
	}

	err := subs.Activate(wg)
	if err != nil {
		return nil, err
	}

	return subs, nil
}

// getRcFile creates a temporary RC file that our shell is initiated from, this allows us to template the logic
// used for initialising the subshell
func getRcFile(v SubShell) (*os.File, error) {
	box := packr.NewBox("../../assets/shells")
	tpl := box.String(v.RcFileTemplate())
	prj := project.Get()

	userScripts := ""
	for _, hook := range prj.Hooks() {
		if hook.Name() == "ACTIVATE" {
			userScripts = userScripts + "\n" + hook.Value()
		}
	}

	inuse := []string{}
	commands := map[string]string{}
	var explicitName string

	// Prepare command map to be parsed by template
	for _, cmd := range prj.Commands() {
		explicitName = fmt.Sprintf("%s_%s", prj.NormalizedName(), cmd.Name())

		_, err := exec.LookPath(cmd.Name())
		if err == nil {
			inuse = append(inuse, cmd.Name())
		}

		commands[cmd.Name()] = cmd.Name()
		commands[explicitName] = cmd.Name()
	}

	// If we have at least one command that's already in use then we should print a warning
	if len(inuse) > 0 {
		print.Warning(locale.Tr("warn_script_name_in_use", strings.Join(inuse, "\n  - "), prj.NormalizedName(), explicitName))
	}

	rcData := map[string]interface{}{
		"Project":     projectfile.Get(),
		"Owner":       prj.Owner(),
		"Name":        prj.Name(),
		"Env":         virtualenvironment.GetEnv(),
		"WD":          virtualenvironment.WorkingDirectory(),
		"UserScripts": userScripts,
		"Commands":    commands,
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

// Get returns the subshell relevant to the current process, but does not activate it
func Get() (SubShell, error) {
	var T = locale.T
	var binary string
	if runtime.GOOS == "windows" {
		binary = os.Getenv("ComSpec")
	} else {
		binary = os.Getenv("SHELL")
	}

	name := filepath.Base(binary)

	var subs SubShell
	switch name {
	case "bash":
		subs = &bash.SubShell{}
	case "zsh":
		subs = &zsh.SubShell{}
	case "tcsh":
		subs = &tcsh.SubShell{}
	case "fish":
		subs = &fish.SubShell{}
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

	return subs, nil
}

// IsActivated returns whether or not this process is being run in an activated
// state.
func IsActivated() bool {
	pid := os.Getppid()
	for true {
		p, err := ps.FindProcess(pid)
		if err != nil {
			logging.Errorf("Could not detect process information: %s", err)
			return false
		}
		if p == nil {
			return false
		}
		if strings.HasPrefix(p.Executable(), constants.CommandName) {
			return true
		}
		ppid := p.PPid()
		if p.PPid() == pid {
			break
		}
		pid = ppid
	}
	return false
}
