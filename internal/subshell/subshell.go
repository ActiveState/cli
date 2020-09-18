package subshell

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/thoas/go-funk"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/osutils"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/process"
	"github.com/ActiveState/cli/internal/subshell/bash"
	"github.com/ActiveState/cli/internal/subshell/cmd"
	"github.com/ActiveState/cli/internal/subshell/fish"
	"github.com/ActiveState/cli/internal/subshell/tcsh"
	"github.com/ActiveState/cli/internal/subshell/zsh"
	"github.com/ActiveState/cli/pkg/project"
)

// SubShell defines the interface for our virtual environment packages, which should be contained in a sub-directory
// under the same directory as this file
type SubShell interface {
	// Activate the given subshell
	Activate(out output.Outputer) *failures.Failure

	// Failures returns a channel to receive failures
	Failures() <-chan *failures.Failure

	// Deactivate the given subshell
	Deactivate() *failures.Failure

	// Run a script string, passing the provided command-line arguments, that assumes this shell and returns the exit code
	Run(filename string, args ...string) error

	// IsActive returns whether the given subshell is active
	IsActive() bool

	// Binary returns the configured binary
	Binary() string

	// SetBinary sets the configured binary, this should only be called by the subshell package
	SetBinary(string)

	// WriteUserEnv writes the given env map to the users environment
	WriteUserEnv(map[string]string, bool) *failures.Failure

	// SetupShellRcFile writes a script or source-able file that updates the environment variables and sets the prompt
	SetupShellRcFile(string, map[string]string, project.Namespaced) error

	// Shell returns an identifiable string representing the shell, eg. bash, zsh
	Shell() string

	// SetEnv sets the environment up for the given subshell
	SetEnv(env map[string]string)

	// SetActivateCommand sets the command that should be ran once activated
	SetActivateCommand(string)

	// Quote will quote the given string, escaping any characters that need escaping
	Quote(value string) string
}

// Get returns the subshell relevant to the current process, but does not activate it
func New() SubShell {
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
		logging.Errorf("Unsupported shell: %s, defaulting to OS default.", name)
		switch runtime.GOOS {
		case "windows":
			return &cmd.SubShell{}
		case "darwin":
			return &zsh.SubShell{}
		default:
			return &bash.SubShell{}
		}
	}

	logging.Debug("Using binary: %s", binary)
	subs.SetBinary(binary)

	env := funk.FilterString(os.Environ(), func(s string) bool {
		return !strings.HasPrefix(s, constants.ProjectEnvVarName)
	})
	subs.SetEnv(osutils.EnvSliceToMap(env))

	return subs
}

// IsActivated returns whether or not this process is being run in an activated
// state.
func IsActivated() bool {
	return process.ActivationPID() != -1
}
