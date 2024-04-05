package packages

import (
	"github.com/ActiveState/cli/internal/captain"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/rtutils/ptr"
	"github.com/ActiveState/cli/internal/runbits/rationalize"
	"github.com/ActiveState/cli/internal/runbits/requirements"
	bpModel "github.com/ActiveState/cli/pkg/platform/api/buildplanner/model"
	"github.com/ActiveState/cli/pkg/platform/model"
)

// UninstallRunParams tracks the info required for running Uninstall.
type UninstallRunParams struct {
	Package captain.PackageValueNoVersion
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

	var nsTypeV *model.NamespaceType
	var ns *model.Namespace

	if params.Package.Namespace != "" {
		ns = ptr.To(model.NewRawNamespace(params.Package.Namespace))
	} else {
		nsTypeV = &nsType
	}

	ts, err := getTime(nil, u.prime.Auth(), u.prime.Project())
	if err != nil {
		return errs.Wrap(err, "Unable to get timestamp from params")
	}

	return requirements.NewRequirementOperation(u.prime).ExecuteRequirementOperation(
		params.Package.Name,
		"",
		nil,
		0, // bit-width placeholder that does not apply here
		bpModel.OperationRemoved,
		ns,
		nsTypeV,
		ts,
	)
}
