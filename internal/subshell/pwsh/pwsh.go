package pwsh

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/osutils"
	"github.com/ActiveState/cli/internal/osutils/user"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/subshell/cmd"
	"github.com/ActiveState/cli/internal/subshell/sscommon"
	"github.com/ActiveState/cli/pkg/project"
)

var escaper *osutils.ShellEscape

func init() {
	escaper = osutils.NewBatchEscaper()
}

// SubShell covers the subshell.SubShell interface, reference that for documentation
type SubShell struct {
	binary string
	rcFile *os.File
	cmd    *exec.Cmd
	env    map[string]string
	errs   chan error
}

const Name string = "powershell"

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
func (v *SubShell) WriteUserEnv(cfg sscommon.Configurable, env map[string]string, envType sscommon.RcIdentification, userScope bool) error {
	cmdShell := &cmd.SubShell{}
	if err := cmdShell.WriteUserEnv(cfg, env, envType, userScope); err != nil {
		return errs.Wrap(err, "Forwarded WriteUserEnv call failed")
	}

	return nil
}

func (v *SubShell) CleanUserEnv(cfg sscommon.Configurable, envType sscommon.RcIdentification, userScope bool) error {
	cmdShell := &cmd.SubShell{}
	if err := cmdShell.CleanUserEnv(cfg, envType, userScope); err != nil {
		return errs.Wrap(err, "Forwarded CleanUserEnv call failed")
	}

	return nil
}

func (v *SubShell) RemoveLegacyInstallPath(_ sscommon.Configurable) error {
	return nil
}

func (v *SubShell) WriteCompletionScript(completionScript string) error {
	return locale.NewError("err_writecompletions_notsupported", "{{.V0}} does not support completions.", v.Shell())
}

func (v *SubShell) RcFile() (string, error) {
	home, err := user.HomeDir()
	if err != nil {
		return "", errs.Wrap(err, "Could not get home dir")
	}

	return filepath.Join(home, "Documents", "WindowsPowerShell", "Microsoft.PowerShell_profile.ps1"), nil
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
	return sscommon.SetupShellRcFile(filepath.Join(targetDir, "shell.ps1"), "pwsh_global.ps1", env, namespace, cfg)
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
func (v *SubShell) Activate(prj *project.Project, cfg sscommon.Configurable, out output.Outputer) error {
	var shellArgs []string
	var directEnv []string

	if prj != nil {
		var err error
		if v.rcFile, err = sscommon.SetupProjectRcFile(prj, "pwsh.ps1", ".ps1", v.env, out, cfg, false); err != nil {
			return err
		}

		shellArgs = []string{"-executionpolicy", "bypass", "-NoExit", "-Command", fmt.Sprintf(". '%s'", v.rcFile.Name())}
	} else {
		directEnv = sscommon.EnvSlice(v.env)
	}

	// powershell -NoExit -Command "& 'C:\Temp\profile.ps1'"
	cmd := sscommon.NewCommand(v.binary, shellArgs, directEnv)
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
	return runtime.GOOS == "windows"
}
