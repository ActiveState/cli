package fish

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/osutils"
	"github.com/ActiveState/cli/internal/osutils/user"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/subshell/sscommon"
	"github.com/ActiveState/cli/pkg/project"
)

var escaper *osutils.ShellEscape

func init() {
	escaper = osutils.NewBashEscaper()
}

// SubShell covers the subshell.SubShell interface, reference that for documentation
type SubShell struct {
	binary string
	rcFile *os.File
	cmd    *exec.Cmd
	env    map[string]string
	errs   chan error
}

const Name string = "fish"

// Shell - see subshell.SubShell
func (v *SubShell) Shell() string {
	return Name
}

// Binary - see subshell.SubShell
func (v *SubShell) Binary() string {
	return v.binary
}

// SetBinary - see subshell.SubShell
func (v *SubShell) SetBinary(binary string) {
	v.binary = binary
}

// WriteUserEnv - see subshell.SubShell
func (v *SubShell) WriteUserEnv(cfg sscommon.Configurable, env map[string]string, envType sscommon.RcIdentification, _ bool) error {
	rcFile, err := v.RcFile()
	if err != nil {
		return errs.Wrap(err, "RcFile failure")
	}

	env = sscommon.EscapeEnv(env)
	return sscommon.WriteRcFile("fishrc_append.fish", rcFile, envType, env)
}

func (v *SubShell) CleanUserEnv(cfg sscommon.Configurable, envType sscommon.RcIdentification, _ bool) error {
	rcFile, err := v.RcFile()
	if err != nil {
		return errs.Wrap(err, "RcFile failure")
	}

	if err := sscommon.CleanRcFile(rcFile, envType); err != nil {
		return errs.Wrap(err, "Failed to remove %s from rcFile", envType)
	}

	return nil
}

func (v *SubShell) RemoveLegacyInstallPath(cfg sscommon.Configurable) error {
	rcFile, err := v.RcFile()
	if err != nil {
		return errs.Wrap(err, "RcFile-failure")
	}

	return sscommon.RemoveLegacyInstallPath(rcFile)
}

func (v *SubShell) WriteCompletionScript(completionScript string) error {
	homeDir, err := user.HomeDir()
	if err != nil {
		return errs.Wrap(err, "IO failure")
	}

	fpath := filepath.Join(homeDir, ".config/fish/completions", constants.CommandName+".fish")
	err = fileutils.WriteFile(fpath, []byte(completionScript))
	if err != nil {
		logging.Debug("Could not write completions script '%s', likely due to non-admin privileges", fpath)
	}

	return nil
}

func (v *SubShell) RcFile() (string, error) {
	homeDir, err := user.HomeDir()
	if err != nil {
		return "", errs.Wrap(err, "IO failure")
	}

	return filepath.Join(homeDir, ".config/fish/config.fish"), nil
}

func (v *SubShell) EnsureRcFileExists() error {
	rcFile, err := v.RcFile()
	if err != nil {
		return errs.Wrap(err, "Could not determine rc file")
	}

	return fileutils.TouchFileUnlessExists(rcFile)
}

// SetupShellRcFile - subshell.SubShell
func (v *SubShell) SetupShellRcFile(targetDir string, env map[string]string, namespace *project.Namespaced, cfg sscommon.Configurable) error {
	env = sscommon.EscapeEnv(env)
	return sscommon.SetupShellRcFile(filepath.Join(targetDir, "shell.fish"), "fishrc_global.fish", env, namespace, cfg)
}

// SetEnv - see subshell.SetEnv
func (v *SubShell) SetEnv(env map[string]string) error {
	v.env = env
	return nil
}

// Quote - see subshell.Quote
func (v *SubShell) Quote(value string) string {
	return escaper.Quote(value)
}

// Activate - see subshell.SubShell
func (v *SubShell) Activate(proj *project.Project, cfg sscommon.Configurable, out output.Outputer) error {
	var shellArgs []string
	var directEnv []string

	// available project files require more intensive modification of shell envs
	if proj != nil {
		env := sscommon.EscapeEnv(v.env)
		var err error
		if v.rcFile, err = sscommon.SetupProjectRcFile(proj, "fishrc.fish", "", env, out, cfg, false); err != nil {
			return err
		}

		shellArgs = append(shellArgs,
			"-i", "-C", fmt.Sprintf("source %s", v.rcFile.Name()),
		)
	} else {
		directEnv = sscommon.EnvSlice(v.env)
	}

	cmd := sscommon.NewCommand(v.Binary(), shellArgs, directEnv)
	v.errs = sscommon.Start(cmd)
	v.cmd = cmd
	return nil
}

// Errors returns a channel for receiving errors related to active behavior
func (v *SubShell) Errors() <-chan error {
	return v.errs
}

// Deactivate - see subshell.SubShell
func (v *SubShell) Deactivate() error {
	if !v.IsActive() {
		return nil
	}

	if err := sscommon.Stop(v.cmd); err != nil {
		return err
	}

	v.cmd = nil
	return nil
}

// Run - see subshell.SubShell
func (v *SubShell) Run(filename string, args ...string) error {
	return sscommon.RunFuncByBinary(v.Binary())(osutils.EnvMapToSlice(v.env), filename, args...)
}

// IsActive - see subshell.SubShell
func (v *SubShell) IsActive() bool {
	return v.cmd != nil && (v.cmd.ProcessState == nil || !v.cmd.ProcessState.Exited())
}

func (v *SubShell) IsAvailable() bool {
	rcFile, err := v.RcFile()
	if err != nil {
		logging.Error("Could not determine rcFile: %s", err)
		return false
	}
	return fileutils.FileExists(rcFile)
}
