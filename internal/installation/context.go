package installation

import (
	"github.com/ActiveState/cli/internal/errs"
)

type Context struct {
	InstalledAsAdmin bool
}

func GetContext() (*Context, error) {
	installedAsAdmin, err := getAdminInstallInformation()
	if err != nil {
		return nil, errs.Wrap(err, "Could not determine if state tool was installed as administrator")
	}

	return &Context{InstalledAsAdmin: installedAsAdmin}, nil
}
