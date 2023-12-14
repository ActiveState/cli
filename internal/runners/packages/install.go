package packages

import (
	"github.com/ActiveState/cli/internal/captain"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/runbits/requirements"
	bpModel "github.com/ActiveState/cli/pkg/platform/api/buildplanner/model"
	"github.com/ActiveState/cli/pkg/platform/model"
)

type PackageVersion struct {
	captain.NameVersion
}

func (pv *PackageVersion) Set(arg string) error {
	err := pv.NameVersion.Set(arg)
	if err != nil {
		return locale.WrapInputError(err, "err_package_format", "The package and version provided is not formatting correctly, must be in the form of <package>@<version>")
	}
	return nil
}

// InstallRunParams tracks the info required for running Install.
type InstallRunParams struct {
	Package PackageVersion
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
	return requirements.NewRequirementOperation(a.prime).ExecuteRequirementOperation(
		params.Package.Name(),
		params.Package.Version(),
		0,
		bpModel.OperationAdded,
		nsType,
	)
}
