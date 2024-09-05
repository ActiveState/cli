package packages

import (
	"github.com/ActiveState/cli/internal/captain"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/rtutils/ptr"
	"github.com/ActiveState/cli/internal/runbits/runtime/requirements"
	"github.com/ActiveState/cli/pkg/platform/api/buildplanner/types"
	"github.com/ActiveState/cli/pkg/platform/model"
)

// InstallRunParams tracks the info required for running Install.
type InstallRunParams struct {
	Packages captain.PackagesValue
	Revision captain.IntValue
}

// Install manages the installing execution context.
type Install struct {
	prime primeable
}

// NewInstall prepares an installation execution context for use.
func NewInstall(prime primeable) *Install {
	return &Install{prime}
}

// Run executes the install behavior.
func (a *Install) Run(params InstallRunParams, nsType model.NamespaceType) (rerr error) {
	defer rationalizeError(a.prime.Auth(), &rerr)

	logging.Debug("ExecuteInstall")
	var reqs []*requirements.Requirement
	for _, p := range params.Packages {
		req := &requirements.Requirement{
			Name:      p.Name,
			Version:   p.Version,
			Operation: types.OperationAdded,
		}

		if p.Namespace != "" {
			req.Namespace = ptr.To(model.NewRawNamespace(p.Namespace))
		} else {
			req.NamespaceType = &nsType
		}

		req.Revision = params.Revision.Int

		reqs = append(reqs, req)
	}

	return requirements.NewRequirementOperation(a.prime).ExecuteRequirementOperation(reqs...)
}
