package bash

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/osutils"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/subshell/sscommon"
	"github.com/ActiveState/cli/pkg/project"
)

var escaper *osutils.ShellEscape

var rcFileName = ".bashrc"

func init() {
	escaper = osutils.NewBashEscaper()
}

// SubShell covers the subshell.SubShell interface, reference that for documentation
type SubShell struct {
	binary          string
	rcFile          *os.File
	cmd             *exec.Cmd
	env             map[string]string
	errs            chan error
	activateCommand *string
}

const Name string = "bash"

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
	return sscommon.WriteRcFile("bashrc_append.sh", rcFile, envType, env)
}

func (v *SubShell) CleanUserEnv(cfg sscommon.Configurable, envType sscommon.RcIdentification, _ bool) error {
	rcFile, err := v.RcFile()
	if err != nil {
		return errs.Wrap(err, "RcFile-failure")
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
	dir := "/usr/local/etc/bash_completion.d/"
	if runtime.GOOS != "darwin" {
		dir = "/etc/bash_completion.d/"
	}

	fpath := filepath.Join(dir, constants.CommandName)
	logging.Debug("Writing to %s: %s", fpath, completionScript)
	err := fileutils.WriteFile(fpath, []byte(completionScript))
	if err != nil {
		return errs.Wrap(err, "Could not write completions script")
	}

	return nil
}

func (v *SubShell) RcFile() (string, error) {
	homeDir, err := fileutils.HomeDir()
	if err != nil {
		return "", errs.Wrap(err, "IO failure")
	}

	rcFilePath := filepath.Join(homeDir, rcFileName)
	// Ensure the .bashrc file exists. On some platforms it is not created by default
	if !fileutils.TargetExists(rcFilePath) {
		err = fileutils.Touch(rcFilePath)
		if err != nil {
			return "", errs.Wrap(err, "Failed to create RCFile at %s", rcFilePath)
		}
	}

	return rcFilePath, nil
}

// SetupShellRcFile - subshell.SubShell
func (v *SubShell) SetupShellRcFile(targetDir string, env map[string]string, namespace project.Namespaced) error {
	env = sscommon.EscapeEnv(env)
	return sscommon.SetupShellRcFile(filepath.Join(targetDir, "shell.sh"), "bashrc_global.sh", env, namespace)
}

// SetEnv - see subshell.SetEnv
func (v *SubShell) SetEnv(env map[string]string) {
	v.env = env
}

// SetActivateCommand - see subshell.SetActivateCommand
func (v *SubShell) SetActivateCommand(cmd string) {
	v.activateCommand = &cmd
}

// Quote - see subshell.Quote
func (v *SubShell) Quote(value string) string {
	return escaper.Quote(value)
}

// Activate - see subshell.SubShell
func (v *SubShell) Activate(proj *project.Project, cfg sscommon.Configurable, out output.Outputer) error {
	var shellArgs []string
	var directEnv []string

	if proj != nil {
		env := sscommon.EscapeEnv(v.env)
		var err error
		if v.rcFile, err = sscommon.SetupProjectRcFile(proj, "bashrc.sh", "", env, out, cfg); err != nil {
			return err
		}

		shellArgs = append(shellArgs, "--rcfile", v.rcFile.Name())
	} else {
		directEnv = sscommon.EnvSlice(v.env)
	}

	if v.activateCommand != nil {
		shellArgs = append(shellArgs, "-c", *v.activateCommand)
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
