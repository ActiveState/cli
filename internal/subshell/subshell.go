package subshell

import (
	"errors"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/rtutils/ptr"
	"github.com/shirou/gopsutil/v3/process"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/multilog"
	"github.com/ActiveState/cli/internal/osutils"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/rollbar"
	"github.com/ActiveState/cli/internal/subshell/bash"
	"github.com/ActiveState/cli/internal/subshell/cmd"
	"github.com/ActiveState/cli/internal/subshell/fish"
	"github.com/ActiveState/cli/internal/subshell/pwsh"
	"github.com/ActiveState/cli/internal/subshell/sscommon"
	"github.com/ActiveState/cli/internal/subshell/tcsh"
	"github.com/ActiveState/cli/internal/subshell/zsh"
	"github.com/ActiveState/cli/pkg/project"
)

const ConfigKeyShell = "shell"

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

	// CleanUserEnv removes the environment setting identified
	CleanUserEnv(sscommon.Configurable, sscommon.RcIdentification, bool) error

	// RemoveLegacyInstallPath removes the install path added to shell configuration by the legacy install scripts
	RemoveLegacyInstallPath(sscommon.Configurable) error

	// WriteCompletionScript writes the completions script for the current shell
	WriteCompletionScript(string) error

	// RcFile return the path of the RC file
	RcFile() (string, error)

	// EnsureRcFile ensures that the RC file exists
	EnsureRcFileExists() error

	// SetupShellRcFile writes a script or source-able file that updates the environment variables and sets the prompt
	SetupShellRcFile(string, map[string]string, *project.Namespaced, sscommon.Configurable) error

	// Shell returns an identifiable string representing the shell, eg. bash, zsh
	Shell() string

	// SetEnv sets the environment up for the given subshell
	SetEnv(env map[string]string) error

	// Quote will quote the given string, escaping any characters that need escaping
	Quote(value string) string

	// IsAvailable returns whether the shell is available on the system
	IsAvailable() bool
}

// New returns the subshell relevant to the current process, but does not activate it
func New(cfg sscommon.Configurable) SubShell {
	name, path := DetectShell(cfg)

	var subs SubShell
	switch name {
	case bash.Name:
		subs = &bash.SubShell{}
	case zsh.Name:
		subs = &zsh.SubShell{}
	case tcsh.Name:
		subs = &tcsh.SubShell{}
	case fish.Name:
		subs = &fish.SubShell{}
	case cmd.Name:
		subs = &cmd.SubShell{}
	case pwsh.Name:
		subs = &pwsh.SubShell{}
	default:
		rollbar.Error("subshell.DetectShell did not return a known name: %s", name)
		switch runtime.GOOS {
		case "windows":
			subs = &cmd.SubShell{}
		case "darwin":
			subs = &zsh.SubShell{}
		default:
			subs = &bash.SubShell{}
		}
	}

	logging.Debug("Using binary: %s", path)
	subs.SetBinary(path)

	err := subs.SetEnv(osutils.EnvSliceToMap(os.Environ()))
	if err != nil {
		// We cannot error here, but this error will resurface when activating a runtime, so we can
		// notify the user at that point.
		logging.Error("Failed to set subshell environment: %v", err)
	}

	return subs
}

// resolveBinaryPath tries to find the named binary on PATH
func resolveBinaryPath(name string) string {
	binaryPath, err := exec.LookPath(name)
	if err == nil {
		// if we found it, resolve all symlinks, for many Linux distributions the SHELL is "sh" but symlinked to a different default shell like bash or zsh
		resolved, err := fileutils.ResolvePath(binaryPath)
		if err == nil {
			return resolved
		} else {
			logging.Debug("Failed to resolve path to shell binary %s: %v", binaryPath, err)
		}
	}
	return name
}

func ConfigureAvailableShells(shell SubShell, cfg sscommon.Configurable, env map[string]string, identifier sscommon.RcIdentification, userScope bool) error {
	// Ensure the given, detected, and current shell has an RC file or else it will not be considered "available"
	err := shell.EnsureRcFileExists()
	if err != nil {
		return errs.Wrap(err, "Could not ensure RC file for current shell")
	}

	for _, s := range supportedShells {
		if !s.IsAvailable() {
			continue
		}
		err := s.WriteUserEnv(cfg, env, identifier, userScope)
		if err != nil {
			logging.Error("Could not update PATH for shell %s: %v", s.Shell(), err)
		}
	}

	return nil
}

// DetectShell detects the shell relevant to the current process and returns its name and path.
func DetectShell(cfg sscommon.Configurable) (string, string) {
	configured := cfg.GetString(ConfigKeyShell)
	var binary string
	defer func() {
		// do not re-write shell binary to config, if the value did not change.
		if configured == binary {
			return
		}
		// We save and use the detected shell to our config so that we can use it when running code through
		// a non-interactive shell
		if err := cfg.Set(ConfigKeyShell, binary); err != nil {
			multilog.Error("Could not save shell binary: %v", errs.JoinMessage(err))
		}
	}()

	if os.Getenv(constants.OverrideShellEnvVarName) != "" {
		binary = os.Getenv(constants.OverrideShellEnvVarName)
	}

	if binary == "" {
		binary = detectShellParent()
	}

	if binary == "" {
		binary = configured
	}

	if binary == "" {
		binary = os.Getenv(SHELL_ENV_VAR)
	}

	if binary == "" {
		binary = OS_DEFAULT
	}

	path := resolveBinaryPath(binary)

	name := filepath.Base(path)
	name = strings.TrimSuffix(name, filepath.Ext(name))
	logging.Debug("Detected SHELL: %s", name)

	if runtime.GOOS == "windows" {
		// For some reason Go or MSYS doesn't translate paths with spaces correctly, so we have to strip out the
		// invalid escape characters for spaces
		path = strings.ReplaceAll(path, `\ `, ` `)
	}

	isKnownShell := false
	for _, ssName := range []string{bash.Name, cmd.Name, fish.Name, tcsh.Name, zsh.Name, pwsh.Name} {
		if name == ssName {
			isKnownShell = true
			break
		}
	}

	if !isKnownShell {
		logging.Debug("Unsupported shell: %s, defaulting to OS default.", name)
		if name != "sh" {
			rollbar.Error("Unsupported shell: %s", name) // we just want to know what this person is using
		}
		switch runtime.GOOS {
		case "windows":
			name = cmd.Name
			path = resolveBinaryPath("cmd.exe")
		case "darwin":
			name = zsh.Name
			path = resolveBinaryPath("zsh")
		default:
			name = bash.Name
			path = resolveBinaryPath("bash")
		}
	}

	return name, path
}

func detectShellParent() string {
	p, err := process.NewProcess(int32(os.Getppid()))
	if err != nil && !errors.As(err, ptr.To(&os.PathError{})) {
		logging.Error("Failed to get parent process: %v", err)
	}

	for p != nil && p.Pid != 0 {
		name, err := p.Name()
		if err == nil {
			if strings.Contains(name, string(filepath.Separator)) {
				name = path.Base(name)
			}
			if supportedShellName(name) {
				return name
			}
		}
		p, _ = p.Parent()
	}

	return ""
}
