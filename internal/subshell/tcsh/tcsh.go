package tcsh

import (
	"os"
	"os/exec"
	"path/filepath"

	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/osutils"
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
	binary          string
	rcFile          *os.File
	cmd             *exec.Cmd
	env             map[string]string
	errs            chan error
	activateCommand *string
}

// Shell - see subshell.SubShell
func (v *SubShell) Shell() string {
	return "tcsh"
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
func (v *SubShell) WriteUserEnv(env map[string]string, envType sscommon.EnvData, _ bool) *failures.Failure {
	homeDir, err := fileutils.HomeDir()
	if err != nil {
		return failures.FailIO.Wrap(err)
	}

	env = sscommon.EscapeEnv(env)
	return sscommon.WriteRcFile("tcshrc_append.sh", filepath.Join(homeDir, ".tcshrc"), envType, env)
}

// SetupShellRcFile - subshell.SubShell
func (v *SubShell) SetupShellRcFile(targetDir string, env map[string]string, namespace project.Namespaced) error {
	env = sscommon.EscapeEnv(env)
	return sscommon.SetupShellRcFile(filepath.Join(targetDir, "shell.tcsh"), "tcsh_global.sh", env, namespace)
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
func (v *SubShell) Activate(out output.Outputer) *failures.Failure {
	// This is horrible but it works.  tcsh doesn't offer a way to override the rc file and
	// doesn't let us run a script and then drop to interactive mode.  So we source the
	// state rc file and then chain an exec which inherits the environment we just set up.
	// It seems to work fine except we don't have a way to override the shell prompt.
	//
	// The exec'd shell does not inherit 'prompt' from the calling terminal since
	// tcsh does not export prompt.  This may be intractable.  I couldn't figure out a
	// hack to make it work.
	env := sscommon.EscapeEnv(v.env)
	var fail *failures.Failure
	if v.rcFile, fail = sscommon.SetupProjectRcFile("tcsh.sh", "", env, out); fail != nil {
		return fail
	}

	shellArgs := []string{"-c", "source " + v.rcFile.Name() + " ; exec " + v.Binary()}
	if v.activateCommand != nil {
		shellArgs[len(shellArgs)-1] = shellArgs[len(shellArgs)-1] + " && " + *v.activateCommand
	}
	cmd := exec.Command(v.Binary(), shellArgs...)

	v.errs = sscommon.Start(cmd)
	v.cmd = cmd
	return nil
}

// Errors returns a channel for receiving errors related to active behavior
func (v *SubShell) Errors() <-chan error {
	return v.errs
}

// Deactivate - see subshell.SubShell
func (v *SubShell) Deactivate() *failures.Failure {
	if !v.IsActive() {
		return nil
	}

	if fail := sscommon.Stop(v.cmd); fail != nil {
		return fail
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
