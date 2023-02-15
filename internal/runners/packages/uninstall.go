package packages

import (
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/runbits/requirements"
	bgModel "github.com/ActiveState/cli/pkg/platform/api/graphql/model/buildplanner"
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
func (u *Uninstall) Run(params UninstallRunParams, nsType model.NamespaceType) error {
	logging.Debug("ExecuteUninstall")
	if u.prime.Project() == nil {
		return locale.NewInputError("err_no_project")
	}

	return requirements.NewRequirementOperation(u.prime).ExecuteRequirementOperation(
		params.Name,
		"",
		0,
		bgModel.OperationRemove,
		nsType,
	)
}
