package cmd

import (
	"os"
	"os/exec"
	"path/filepath"

	"github.com/spf13/viper"

	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/osutils"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/subshell/sscommon"
	"github.com/ActiveState/cli/pkg/project"
)

var escaper *osutils.ShellEscape

func init() {
	escaper = osutils.NewBatchEscaper()
}

// SubShell covers the subshell.SubShell interface, reference that for documentation
type SubShell struct {
	binary          string
	rcFile          *os.File
	cmd             *exec.Cmd
	env             map[string]string
	fs              chan *failures.Failure
	activateCommand *string
}

// Shell - see subshell.SubShell
func (v *SubShell) Shell() string {
	return "cmd"
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
func (v *SubShell) WriteUserEnv(env map[string]string, envType sscommon.EnvType, userScope bool) *failures.Failure {
	cmdEnv := NewCmdEnv(userScope)

	// Clean up old entries
	oldEnv := viper.GetStringMap("user_env")
	for k, v := range oldEnv {
		if fail := cmdEnv.unset(k, v.(string)); fail != nil {
			return fail
		}
	}

	// Store new entries
	viper.Set("user_env", env)

	for k, v := range env {
		value := v
		if k == "PATH" {
			path, fail := cmdEnv.get("PATH")
			if fail != nil {
				return fail
			}
			if path != "" {
				path = ";" + path
			}

			value = v + path
		}

		// Set key/value in the user environment
		fail := cmdEnv.set(k, value)
		if fail != nil {
			return fail
		}
	}

	cmdEnv.propagate()
	return nil
}

// SetupShellRcFile - subshell.SubShell
func (v *SubShell) SetupShellRcFile(targetDir string, env map[string]string, namespace project.Namespaced) error {
	env = sscommon.EscapeEnv(env)
	return sscommon.SetupShellRcFile(filepath.Join(targetDir, "shell.bat"), "config_global.bat", env, namespace)
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
	env := sscommon.EscapeEnv(v.env)
	var fail *failures.Failure
	if v.rcFile, fail = sscommon.SetupProjectRcFile("config.bat", ".bat", env, out); fail != nil {
		return fail
	}

	shellArgs := []string{"/K", v.rcFile.Name()}
	if v.activateCommand != nil {
		if fail := fileutils.AppendToFile(v.rcFile.Name(), []byte("\r\n"+*v.activateCommand+"\r\nexit")); fail != nil {
			return fail
		}
	}

	cmd := exec.Command("cmd", shellArgs...)

	v.fs = sscommon.Start(cmd)
	v.cmd = cmd
	return nil
}

// Failures returns a channel for receiving errors related to active behavior
func (v *SubShell) Failures() <-chan *failures.Failure {
	return v.fs
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
