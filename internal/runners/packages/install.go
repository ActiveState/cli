package packages

import (
	"github.com/ActiveState/cli/internal/captain"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/rtutils/p"
	"github.com/ActiveState/cli/internal/runbits/requirements"
	bgModel "github.com/ActiveState/cli/pkg/platform/api/graphql/model/buildplanner"
	"github.com/ActiveState/cli/pkg/platform/model"
)

// InstallRunParams tracks the info required for running Install.
type InstallRunParams struct {
	Package   captain.PackageFlag
	Timestamp captain.TimeFlag
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
func (a *Install) Run(params InstallRunParams, nsType model.NamespaceType) error {
	var nsTypeV *model.NamespaceType
	var ns *model.Namespace

	logging.Debug("ExecuteInstall")
	if params.Package.Namespace != "" {
		ns = p.Pointer(model.NewRawNamespace(params.Package.Namespace))
	} else {
		nsTypeV = &nsType
	}

	return requirements.NewRequirementOperation(a.prime).ExecuteRequirementOperation(
		params.Package.Name,
		params.Package.Version,
		bgModel.OperationAdd,
		ns,
		nsTypeV,
		params.Timestamp.Time,
	)
}
