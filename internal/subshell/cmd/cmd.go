package cmd

import (
	"os"
	"os/exec"

	"github.com/spf13/viper"

	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/internal/osutils"
	"github.com/ActiveState/cli/internal/subshell/sscommon"
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
	fs     chan *failures.Failure
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
func (v *SubShell) WriteUserEnv(env map[string]string, systemEnv bool) *failures.Failure {
	cmdEnv := NewCmdEnv(systemEnv)

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

// SetEnv - see subshell.SetEnv
func (v *SubShell) SetEnv(env map[string]string) {
	v.env = env
}

// Quote - see subshell.Quote
func (v *SubShell) Quote(value string) string {
	return escaper.Quote(value)
}

// Activate - see subshell.SubShell
func (v *SubShell) Activate() *failures.Failure {
	env := sscommon.EscapeEnv(v.env)
	var fail *failures.Failure
	if v.rcFile, fail = sscommon.SetupProjectRcFile("config.bat", ".bat", env); fail != nil {
		return fail
	}

	shellArgs := []string{"/K", v.rcFile.Name()}

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
