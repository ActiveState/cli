package cmd

import (
	"errors"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/spf13/viper"
	"golang.org/x/sys/windows/registry"
	"os"
	"os/exec"

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
	env    []string
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
func (v *SubShell) WriteUserEnv(env map[string]string) error {
	// Clean up old entries
	oldEnv := viper.GetStringMap("user_env")
	for k, v  := range oldEnv {
		if err := unsetUserEnv(k, v.(string)); err != nil {
			return err
		}
	}

	// Store new entries
	viper.Set("user_env", env)

	for k, v := range env {
		value := v
		if k == "PATH" {
			path, err := getUserEnv("PATH")
			if err != nil {
				return err
			}
			if path != "" {
				path = ";" + path
			}

			value = v+path
		}

		// Set key/value in the user environment
		err := setUserEnv(k, value)
		if err != nil {
			return err
		}
	}
	return nil
}

func unsetUserEnv(name, ifValueEquals string) error {
	key, err := registry.OpenKey(registry.CURRENT_USER, "Environment", registry.ALL_ACCESS)
	if err != nil {
		return failures.FailOS.Wrap(err, locale.T("err_windows_registry"))
	}
	defer key.Close()

	v, _, err := key.GetStringValue(name)
	if err != nil {
		if errors.Is(err, registry.ErrNotExist) {
			return nil
		}
		return failures.FailOS.Wrap(err, locale.T("err_windows_registry"))
	}

	if v != ifValueEquals {
		return nil
	}

	// Check for backup value
	backupValue, _, err := key.GetStringValue(envBackupName(name))
	realError := err != nil && ! errors.Is(err, registry.ErrNotExist)
	backupExists := err == nil

	if realError {
		return failures.FailOS.Wrap(err, locale.T("err_windows_registry"))
	}
	if backupExists {
		if err := key.DeleteValue(envBackupName(name)); err != nil {
			return failures.FailOS.Wrap(err, locale.T("err_windows_registry"))
		}
		return key.SetStringValue(name, backupValue)
	}
	return key.DeleteValue(name)
}


func setUserEnv(name, newValue string) error {
	key, err := registry.OpenKey(registry.CURRENT_USER, "Environment", registry.ALL_ACCESS)
	if err != nil {
		return failures.FailOS.Wrap(err, locale.T("err_windows_registry"))
	}
	defer key.Close()

	// Check if we're going to be overriding
	oldValue, _, err := key.GetStringValue(name)
	if err != nil && ! errors.Is(err, registry.ErrNotExist) {
		return failures.FailOS.Wrap(err, locale.T("err_windows_registry"))
	} else if err == nil {
		// Save backup
		if err2 := key.SetStringValue(envBackupName(name), oldValue); err2 != nil {
			return failures.FailOS.Wrap(err2, locale.T("err_windows_registry"))
		}
	}

	return key.SetStringValue(name, newValue)
}

func getUserEnv(name string) (string, error) {
	key, err := registry.OpenKey(registry.CURRENT_USER, "Environment", registry.ALL_ACCESS)
	if err != nil {
		return "", failures.FailOS.Wrap(err, locale.T("err_windows_registry"))
	}
	defer key.Close()

	// Check if we're going to be overriding
	originalValue, _, err := key.GetStringValue(envBackupName(name))
	if err != nil && ! errors.Is(err, registry.ErrNotExist) {
		return "", failures.FailOS.Wrap(err, locale.T("err_windows_registry"))
	} else if err == nil {
		return originalValue, nil
	}

	v, _, err := key.GetStringValue(name)
	if err != nil && ! errors.Is(err, registry.ErrNotExist) {
		return v, failures.FailOS.Wrap(err, locale.T("err_windows_registry"))
	}
	return v, nil
}

func envBackupName(name string) string {
	return name+"_ORIGINAL"
}

// SetEnv - see subshell.SetEnv
func (v *SubShell) SetEnv(env []string) {
	v.env = env
}

// Quote - see subshell.Quote
func (v *SubShell) Quote(value string) string {
	return escaper.Quote(value)
}

// Activate - see subshell.SubShell
func (v *SubShell) Activate() *failures.Failure {
	var fail *failures.Failure
	if v.rcFile, fail = sscommon.SetupProjectRcFile("config.bat", ".bat"); fail != nil {
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
	return sscommon.RunFuncByBinary(v.Binary())(v.env, filename, args...)
}

// IsActive - see subshell.SubShell
func (v *SubShell) IsActive() bool {
	return v.cmd != nil && (v.cmd.ProcessState == nil || !v.cmd.ProcessState.Exited())
}
