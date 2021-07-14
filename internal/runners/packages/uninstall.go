package packages

import (
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/pkg/platform/model"
)

// UninstallRunParams tracks the info required for running Uninstall.
type UninstallRunParams struct {
	Name string
}

// Uninstall manages the uninstalling execution context.
type Uninstall struct {
	prime primeable
}

// NewUninstall prepares an uninstallation execution context for use.
func NewUninstall(prime primeable) *Uninstall {
	return &Uninstall{prime}
}

// Run executes the uninstall behavior.
func (r *Uninstall) Run(params UninstallRunParams, nstype model.NamespaceType) error {
	logging.Debug("ExecuteUninstall")
	if r.prime.Project() == nil {
		return locale.NewInputError("err_no_project")
	}

	return executePackageOperation(r.prime, params.Name, "", model.OperationRemoved, nstype)
}
