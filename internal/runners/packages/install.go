package packages

import (
	"github.com/ActiveState/cli/internal/captain"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/runbits/requirements"
	bgModel "github.com/ActiveState/cli/pkg/platform/api/graphql/model/buildplanner"
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
	Package   PackageVersion
	Language  string
	Namespace string
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
	logging.Debug("ExecuteInstall")
	if params.Namespace != "" {
		var err error
		nsType, err = model.NewNamespaceType(params.Namespace)
		if err != nil {
			return locale.WrapError(err, "err_namespace_type", "Could not determine namespace type")
		}
	}

	return requirements.NewRequirementOperation(a.prime).ExecuteRequirementOperation(
		params.Package.Name(),
		params.Package.Version(),
		params.Language,
		0,
		bgModel.OperationAdd,
		nsType,
	)
}
