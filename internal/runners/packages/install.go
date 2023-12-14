package packages

import (
	"github.com/ActiveState/cli/internal/captain"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/rtutils/ptr"
	"github.com/ActiveState/cli/internal/runbits/requirements"
	bpModel "github.com/ActiveState/cli/pkg/platform/api/buildplanner/model"
	"github.com/ActiveState/cli/pkg/platform/model"
)

// InstallRunParams tracks the info required for running Install.
type InstallRunParams struct {
	Package   captain.PackageValue
	Timestamp captain.TimeValue
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
	var nsTypeV *model.NamespaceType
	var ns *model.Namespace

	logging.Debug("ExecuteInstall")
	if params.Package.Namespace != "" {
		ns = ptr.To(model.NewRawNamespace(params.Package.Namespace))
	} else {
		nsTypeV = &nsType
	}

	return requirements.NewRequirementOperation(a.prime).ExecuteRequirementOperation(
		params.Package.Name,
		params.Package.Version,
		bpModel.OperationAdded,
		ns,
		nsTypeV,
		params.Timestamp.Time,
	)
}
