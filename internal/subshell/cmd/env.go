package cmd

import (
	"log"

	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/internal/locale"
)

type RegistryKey interface {
	GetStringValue(name string) (val string, valtype uint32, err error)
	SetStringValue(name, value string) error
	DeleteValue(name string) error
	Close() error
}

type OpenKeyFn func(path string) (RegistryKey, error)

type CmdEnv struct {
	openKeyFn OpenKeyFn
}

func NewCmdEnv() *CmdEnv {
	return &CmdEnv{OpenKey}
}

// unsetUserEnv clears a state cool configured environment variable
// It only does this if the value equals the expected value (meaning if we can verify that state tool was in fact
// responsible for setting it)
func (c *CmdEnv) unset(name, ifValueEquals string) *failures.Failure {
	key, err := c.openKeyFn("Environment")
	if err != nil {
		return failures.FailOS.Wrap(err, locale.T("err_windows_registry"))
	}
	defer key.Close()

	v, _, err := key.GetStringValue(name)
	if err != nil {
		if IsNotExistError(err) {
			return nil
		}
		return failures.FailOS.Wrap(err, locale.T("err_windows_registry"))
	}

	// Check if we are responsible for the value
	if v != ifValueEquals {
		return nil
	}

	// Check for backup value
	backupValue, _, err := key.GetStringValue(envBackupName(name))
	realError := err != nil && ! IsNotExistError(err)
	backupExists := err == nil

	if realError {
		return failures.FailOS.Wrap(err, locale.T("err_windows_registry"))
	}
	if backupExists {
		// If a backup exists (ie. the value before we modified it) then restore that rather than deleting it altogether
		if err := key.DeleteValue(envBackupName(name)); err != nil {
			return failures.FailOS.Wrap(err, locale.T("err_windows_registry"))
		}
		return failures.FailOS.Wrap(key.SetStringValue(name, backupValue))
	}
	return failures.FailOS.Wrap(key.DeleteValue(name))
}

// setUserEnv sets a variable in the user environment and saves the original as a backup
func (c *CmdEnv) set(name, newValue string) *failures.Failure {
	key, err := c.openKeyFn("Environment")
	if err != nil {
		return failures.FailOS.Wrap(err, locale.T("err_windows_registry"))
	}
	defer key.Close()

	// Check if we're going to be overriding
	oldValue, _, err := key.GetStringValue(name)
	if err != nil && ! IsNotExistError(err) {
		return failures.FailOS.Wrap(err, locale.T("err_windows_registry"))
	} else if err == nil {
		// Save backup
		if err2 := key.SetStringValue(envBackupName(name), oldValue); err2 != nil {
			return failures.FailOS.Wrap(err2, locale.T("err_windows_registry"))
		}
	}

	return failures.FailOS.Wrap(key.SetStringValue(name, newValue))
}

// getUserEnv retrieves a variable from the user environment, this prioritizes a backup if it exists
func (c *CmdEnv) get(name string) (string, *failures.Failure) {
	key, err := c.openKeyFn("Environment")
	if err != nil {
		return "", failures.FailOS.Wrap(err, locale.T("err_windows_registry"))
	}
	defer key.Close()

	// Return the backup version if it exists
	originalValue, _, err := key.GetStringValue(envBackupName(name))
	if err != nil && ! IsNotExistError(err) {
		return "", failures.FailOS.Wrap(err, locale.T("err_windows_registry"))
	} else if err == nil {
		return originalValue, nil
	}

	v, _, err := key.GetStringValue(name)
	if err != nil && ! IsNotExistError(err) {
		return v, failures.FailOS.Wrap(err, locale.T("err_windows_registry"))
	}
	return v, nil
}

// GetUnsafe is an alias for `get` intended for use by tests/integration tests, don't use for anything else!
func (c *CmdEnv) GetUnsafe(name string) string {
	r, f := c.get(name)
	if f != nil {
		log.Fatalf("GetUnsafe failed with: %s", f.Error())
	}
	return r
}

func envBackupName(name string) string {
	return name + "_ORIGINAL"
}
