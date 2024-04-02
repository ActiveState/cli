package packages

import (
	"github.com/ActiveState/cli/internal/captain"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/rtutils/ptr"
	"github.com/ActiveState/cli/internal/runbits/rationalize"
	"github.com/ActiveState/cli/internal/runbits/requirements"
	bpModel "github.com/ActiveState/cli/pkg/platform/api/buildplanner/model"
	"github.com/ActiveState/cli/pkg/platform/model"
)

// UninstallRunParams tracks the info required for running Uninstall.
type UninstallRunParams struct {
	Packages captain.PackagesValue
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
func (u *Uninstall) Run(params UninstallRunParams, nsType model.NamespaceType) (rerr error) {
	defer rationalizeError(u.prime.Auth(), &rerr)
	logging.Debug("ExecuteUninstall")
	if u.prime.Project() == nil {
		return rationalize.ErrNoProject
	}

	var reqs []*requirements.Requirement
	for _, p := range params.Packages {
		req := &requirements.Requirement{
			Name:      p.Name,
			Version:   p.Version,
			Operation: bpModel.OperationRemoved,
		}

		if p.Namespace != "" {
			req.Namespace = ptr.To(model.NewRawNamespace(p.Namespace))
		} else {
			req.NamespaceType = &nsType
		}

		reqs = append(reqs, req)
	}

	return requirements.NewRequirementOperation(u.prime).ExecuteRequirementOperation(nil, reqs...)
}
