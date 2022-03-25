package installation

import (
	"fmt"
	"os/user"
	"strconv"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/osutils"
)

const (
	// adminInstallRegistry is the registry key name used to determine if the State Tool was installed as administrator
	adminInstallRegistry = "Installed As Admin"

	installRegistryKeyPath = `SOFTWARE\ActiveState\install`
)

func SaveContext(context *Context) error {
	user, err := user.Current()
	if err != nil {
		return errs.Wrap(err, "Could not get current user")
	}

	key, _, err := osutils.CreateUserKey(fmt.Sprintf(`%s\%s`, user.Uid, installRegistryKeyPath))
	if err != nil {
		return errs.Wrap(err, "Could not create registry key")
	}
	defer key.Close()

	err = key.SetStringValue(adminInstallRegistry, strconv.FormatBool(context.InstalledAsAdmin))
	if err != nil {
		return errs.Wrap(err, "Could not set registry key value")
	}

	return nil
}

func getAdminInstallInformation() (bool, error) {
	key, err := osutils.OpenUserKey(installRegistryKeyPath)
	if err != nil {
		return false, errs.Wrap(err, "Could not get key value")
	}
	defer key.Close()

	v, _, err := key.GetStringValue(adminInstallRegistry)
	if err != nil {
		return false, errs.Wrap(err, "Could not get string value")
	}

	installedAsAdmin, err := strconv.ParseBool(v)
	if err != nil {
		return false, errs.Wrap(err, "Could not parse bool from string value: %s", v)
	}

	return installedAsAdmin, nil
}
