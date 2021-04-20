package subshell

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/thoas/go-funk"

	"github.com/ActiveState/cli/internal/config"
	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/osutils"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/subshell/bash"
	"github.com/ActiveState/cli/internal/subshell/cmd"
	"github.com/ActiveState/cli/internal/subshell/fish"
	"github.com/ActiveState/cli/internal/subshell/sscommon"
	"github.com/ActiveState/cli/internal/subshell/tcsh"
	"github.com/ActiveState/cli/internal/subshell/zsh"
	"github.com/ActiveState/cli/pkg/project"
)

// SubShell defines the interface for our virtual environment packages, which should be contained in a sub-directory
// under the same directory as this file
type SubShell interface {
	// Activate the given subshell
	Activate(proj *project.Project, cfg sscommon.Configurable, out output.Outputer) error

	// Errors returns a channel to receive errors
	Errors() <-chan error

	// Deactivate the given subshell
	Deactivate() error

	// Run a script string, passing the provided command-line arguments, that assumes this shell and returns the exit code
	Run(filename string, args ...string) error

	// IsActive returns whether the given subshell is active
	IsActive() bool

	// Binary returns the configured binary
	Binary() string

	// SetBinary sets the configured binary, this should only be called by the subshell package
	SetBinary(string)

	// WriteUserEnv writes the given env map to the users environment
	WriteUserEnv(sscommon.Configurable, map[string]string, sscommon.RcIdentification, bool) error

	// WriteCompletionScript writes the completions script for the current shell
	WriteCompletionScript(string) error

	// RcFile return the path of the RC file
	RcFile() (string, error)

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

// New returns the subshell relevant to the current process, but does not activate it
func New(cfg *config.Instance) SubShell {
	binary := DetectShellBinary(cfg)

	// try to find the binary on the PATH
	binaryPath, err := exec.LookPath(binary)
	if err == nil {
		// if we found it, resolve all symlinks, for many Linux distributions the SHELL is "sh" but symlinked to a different default shell like bash or zsh
		resolved, err := fileutils.ResolvePath(binaryPath)
		if err == nil {
			binary = resolved
		} else {
			logging.Debug("Failed to resolve path to shell binary %s: %v", binaryPath, err)
		}
	}

	name := filepath.Base(binary)
	name = strings.TrimSuffix(name, filepath.Ext(name))
	logging.Debug("Detected SHELL: %s", name)

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
		logging.Debug("Unsupported shell: %s, defaulting to OS default.", name)
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

func DetectShellBinary(cfg *config.Instance) (binary string) {
	configured := cfg.GetString(config.ConfigKeyShell)
	defer func() {
		// do not re-write shell binary to config, if the value did not change.
		if configured == binary {
			return
		}
		// We save and use the detected shell to our config so that we can use it when running code through
		// a non-interactive shell
		if err := cfg.Set(config.ConfigKeyShell, binary); err != nil {
			logging.Error("Could not save shell binary: %v", errs.Join(err, ": "))
		}
	}()

	if binary := os.Getenv("SHELL"); binary != "" {
		return binary
	}

	if runtime.GOOS == "windows" {
		binary = os.Getenv("ComSpec")
		if binary != "" {
			return binary
		}
	}

	fallback := configured
	if fallback == "" {
		if runtime.GOOS == "windows" {
			fallback = "cmd.exe"
		} else {
			fallback = "bash"
		}
	}

	return fallback
}
