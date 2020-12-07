package cmd

import (
	"log"

	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/osutils"
)

type OpenKeyFn func(path string) (osutils.RegistryKey, error)

type CmdEnv struct {
	openKeyFn OpenKeyFn
	// whether this updates the system environment
	userScope bool
}

func NewCmdEnv(userScope bool) *CmdEnv {
	openKeyFn := osutils.OpenSystemKey
	if userScope {
		openKeyFn = osutils.OpenUserKey
	}
	return &CmdEnv{
		openKeyFn: openKeyFn,
		userScope: userScope,
	}
}

func getEnvironmentPath(userScope bool) string {
	if userScope {
		return "Environment"
	}
	return `SYSTEM\ControlSet001\Control\Session Manager\Environment`
}

// unsetUserEnv clears a state cool configured environment variable
// It only does this if the value equals the expected value (meaning if we can verify that state tool was in fact
// responsible for setting it)
func (c *CmdEnv) unset(name, ifValueEquals string) *failures.Failure {
	key, err := c.openKeyFn(getEnvironmentPath(c.userScope))
	if err != nil {
		return failures.FailOS.Wrap(err, locale.T("err_windows_registry"))
	}
	defer key.Close()

	v, _, err := key.GetStringValue(name)
	if err != nil {
		if osutils.IsNotExistError(err) {
			return nil
		}
		return failures.FailOS.Wrap(err, locale.T("err_windows_registry"))
	}

	// Check if we are responsible for the value
	if v != ifValueEquals {
		return nil
	}

	// Delete value
	return failures.FailOS.Wrap(key.DeleteValue(name))
}

// setUserEnv sets a variable in the user environment and saves the original as a backup
func (c *CmdEnv) set(name, newValue string) *failures.Failure {
	key, err := c.openKeyFn(getEnvironmentPath(c.userScope))
	if err != nil {
		return failures.FailOS.Wrap(err, locale.T("err_windows_registry"))
	}
	defer key.Close()

	// Check if we're going to be overriding
	_, valType, err := key.GetStringValue(name)
	if err != nil && !osutils.IsNotExistError(err) {
		return failures.FailOS.Wrap(err, locale.T("err_windows_registry"))
	}

	return failures.FailOS.Wrap(osutils.SetStringValue(key, name, valType, newValue))
}

// Get retrieves a variable from the user environment, this prioritizes a backup if it exists
func (c *CmdEnv) Get(name string) (string, *failures.Failure) {
	key, err := c.openKeyFn(getEnvironmentPath(c.userScope))
	if err != nil {
		return "", failures.FailOS.Wrap(err, locale.T("err_windows_registry"))
	}
	defer key.Close()

	v, _, err := key.GetStringValue(name)
	if err != nil && !osutils.IsNotExistError(err) {
		return v, failures.FailOS.Wrap(err, locale.T("err_windows_registry"))
	}
	return v, nil
}

// GetUnsafe is an alias for `get` intended for use by tests/integration tests, don't use for anything else!
func (c *CmdEnv) GetUnsafe(name string) string {
	r, f := c.Get(name)
	if f != nil {
		log.Fatalf("GetUnsafe failed with: %s", f.Error())
	}
	return r
}
