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

	"github.com/alecthomas/template"
	"github.com/gobuffalo/packr"
	tempfile "github.com/mash/go-tempfile-suffix"
	ps "github.com/mitchellh/go-ps"
	funk "github.com/thoas/go-funk"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/print"
	"github.com/ActiveState/cli/internal/subshell/bash"
	"github.com/ActiveState/cli/internal/subshell/cmd"
	"github.com/ActiveState/cli/internal/subshell/fish"
	"github.com/ActiveState/cli/internal/subshell/tcsh"
	"github.com/ActiveState/cli/internal/subshell/zsh"
	"github.com/ActiveState/cli/internal/virtualenvironment"
	"github.com/ActiveState/cli/pkg/project"
)

// SubShell defines the interface for our virtual environment packages, which should be contained in a sub-directory
// under the same directory as this file
type SubShell interface {
	// Activate the given subshell
	Activate(wg *sync.WaitGroup) error

	// Deactivate the given subshell
	Deactivate() error

	// Run a script string, passing the provided command-line arguments, that assumes this shell and returns the exit code
	Run(script string, args ...string) (int, error)

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

	// SetEnv sets the environment up for the given subshell
	SetEnv(env []string)

	// Quote will quote the given string, escaping any characters that need escaping
	Quote(value string) string
}

// Activate the virtual environment
func Activate(wg *sync.WaitGroup) (SubShell, error) {
	logging.Debug("Activating Subshell")

	// Why another check here? Because some things like events / run script don't take the virtualenv route,
	// realistically this shouldn't really happen, but it's a useful failsafe for us
	activeProject := os.Getenv(constants.ActivatedStateEnvVarName)
	if activeProject != "" {
		return nil, virtualenvironment.FailAlreadyActive.New("err_already_active")
	}

	subs, fail := Get()
	if fail != nil {
		return nil, fail
	}

	logging.Debug("Calling Activate")
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
	for _, event := range prj.Events() {
		if event.Name() == "ACTIVATE" {
			userScripts = userScripts + "\n" + event.Value()
		}
	}

	inuse := []string{}
	scripts := map[string]string{}
	var explicitName string

	// Prepare script map to be parsed by template
	for _, cmd := range prj.Scripts() {
		explicitName = fmt.Sprintf("%s_%s", prj.NormalizedName(), cmd.Name())

		_, err := exec.LookPath(cmd.Name())
		if err == nil {
			inuse = append(inuse, cmd.Name())
		}

		scripts[cmd.Name()] = cmd.Name()
		scripts[explicitName] = cmd.Name()
	}

	// If we have at least one script that's already in use then we should print a warning
	if len(inuse) > 0 {
		print.Warning(locale.Tr("warn_script_name_in_use", strings.Join(inuse, "\n  - "), prj.NormalizedName(), explicitName))
	}

	rcData := map[string]interface{}{
		"Owner":       prj.Owner(),
		"Name":        prj.Name(),
		"Env":         virtualenvironment.Get().GetEnv(),
		"WD":          virtualenvironment.Get().WorkingDirectory(),
		"UserScripts": userScripts,
		"Scripts":     scripts,
	}
	t, err := template.New("rcfile").Parse(tpl)
	if err != nil {
		return nil, err
	}

	var out bytes.Buffer
	err = t.Execute(&out, rcData)
	if err != nil {
		return nil, err
	}

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
	binary := os.Getenv("SHELL")
	if binary == "" {
		if runtime.GOOS == "windows" {
			binary = os.Getenv("ComSpec")
			if binary == "" {
				binary = "cmd.exe"
			}
		} else {
			binary = "bash"
		}
	}

	logging.Debug("Detected SHELL: %s", binary)

	name := filepath.Base(binary)
	name = strings.TrimSuffix(name, filepath.Ext(name))

	if runtime.GOOS == "windows" {
		// For some reason Go or MSYS doesn't translate paths with spaces correctly, so we have to strip out the
		// invalid escape characters for spaces
		binary = strings.ReplaceAll(binary, `\ `, ` `)
	}

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
	case "cmd":
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

	logging.Debug("Using binary: %s", binary)
	subs.SetBinary(binary)
	logging.Debug("Using RC File: %s", rcFile.Name())
	subs.SetRcFile(rcFile)
	os.Setenv("BASH_ENV", rcFile.Name())

	env := funk.FilterString(os.Environ(), func(s string) bool {
		return !strings.HasPrefix(s, constants.ProjectEnvVarName)
	})
	subs.SetEnv(env)

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
