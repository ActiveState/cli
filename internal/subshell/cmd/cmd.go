package cmd

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
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
	binary string
	rcFile *os.File
	cmd    *exec.Cmd
	env    map[string]string
	errs   chan error
}

const Name string = "cmd"

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
	cmdEnv := NewCmdEnv(userScope)

	// Clean up old entries
	oldEnv := cfg.GetStringMap(envType.Key)
	for k, v := range oldEnv {
		if err := cmdEnv.unset(k, v.(string)); err != nil {
			return err
		}
	}

	// Store new entries
	err := cfg.Set(envType.Key, env)
	if err != nil {
		return errs.Wrap(err, "Could not set env infomation in config")
	}

	for k, v := range env {
		value := v
		if k == "PATH" {
			path, err := cmdEnv.Get("PATH")
			if err != nil {
				return err
			}
			if path != "" {
				path = ";" + path
			}

			value = v + path
		}

		// Set key/value in the user environment
		err := cmdEnv.Set(k, value)
		if err != nil {
			return err
		}
	}

	if err := osutils.PropagateEnv(); err != nil {
		return errs.Wrap(err, "Sending OS signal to update environment failed.")
	}
	return nil
}

func (v *SubShell) CleanUserEnv(cfg sscommon.Configurable, envType sscommon.RcIdentification, userScope bool) error {
	cmdEnv := NewCmdEnv(userScope)

	// Clean up old entries
	oldEnv := cfg.GetStringMap(envType.Key)
	for k, v := range oldEnv {
		val, ok := v.(string)
		if !ok {
			logging.Debug("Invalid non-string value in environment mapping")
			continue
		}
		if err := cmdEnv.unset(k, val); err != nil {
			return err
		}
	}

	if err := osutils.PropagateEnv(); err != nil {
		return errs.Wrap(err, "Sending OS signal to update environment failed.")
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
	return "", locale.NewError("err_cmd_rcile", "cmd does not support RC files")
}

func (v *SubShell) EnsureRcFileExists() error {
	// Windows does not use RC files
	return nil
}

// SetupShellRcFile - subshell.SubShell
func (v *SubShell) SetupShellRcFile(targetDir string, env map[string]string, namespace *project.Namespaced, cfg sscommon.Configurable) error {
	env = sscommon.EscapeEnv(env)
	return sscommon.SetupShellRcFile(filepath.Join(targetDir, "shell.bat"), "config_global.bat", env, namespace, cfg)
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
		env := sscommon.EscapeEnv(v.env)
		var err error
		if v.rcFile, err = sscommon.SetupProjectRcFile(prj, "config.bat", ".bat", env, out, cfg, false); err != nil {
			return err
		}

		shellArgs = append(shellArgs, "/K", v.rcFile.Name())
	} else {
		directEnv = sscommon.EnvSlice(v.env)
	}

	cmd := sscommon.NewCommand("cmd", shellArgs, directEnv)
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
