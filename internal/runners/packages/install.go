package packages

import (
	"time"

	"github.com/ActiveState/cli/internal/captain"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/rtutils/ptr"
	"github.com/ActiveState/cli/internal/runbits/requirements"
	"github.com/ActiveState/cli/pkg/localcommit"
	bpModel "github.com/ActiveState/cli/pkg/platform/api/buildplanner/model"
	"github.com/ActiveState/cli/pkg/platform/authentication"
	"github.com/ActiveState/cli/pkg/platform/model"
	"github.com/ActiveState/cli/pkg/project"
)

// InstallRunParams tracks the info required for running Install.
type InstallRunParams struct {
	Package   captain.PackageValue
	Timestamp captain.TimeValue
	Revision  captain.IntValue
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

	ts, err := getTime(&params.Timestamp, a.prime.Auth(), a.prime.Project())
	if err != nil {
		return errs.Wrap(err, "Unable to get timestamp from params")
	}

	return requirements.NewRequirementOperation(a.prime).ExecuteRequirementOperation(
		params.Package.Name,
		params.Package.Version,
		params.Revision.Int,
		0, // bit-width placeholder that does not apply here
		bpModel.OperationAdded,
		ns,
		nsTypeV,
		ts,
	)
}

// getTime returns a timestamp based on the given "--ts" value.
// If "now" was given, returns "now" according to the platform.
// If no timestamp was given but the current project's commit time is after the latest inventory
// timestamp, returns that commit time.
// Otherwise, returns the specified timestamp or nil (which falls back on the default Platform
// timestamp for a given operation)
func getTime(ts *captain.TimeValue, auth *authentication.Auth, proj *project.Project) (*time.Time, error) {
	if ts.Now {
		latest, err := model.FetchLatestRevisionTimeStamp(auth)
		if err != nil {
			return nil, errs.Wrap(err, "Unable to determine latest revision time")
		}
		return &latest, nil
	}

	if ts.Time == nil && proj != nil {
		latest, err := model.FetchLatestTimeStamp()
		if err != nil {
			return nil, errs.Wrap(err, "Unable to fetch latest Platform timestamp")
		}

		commitID, err := localcommit.Get(proj.Dir())
		if err != nil {
			return nil, errs.Wrap(err, "Unable to get commit ID")
		}

		atTime, err := model.FetchTimeStampForCommit(commitID)
		if err != nil {
			return nil, errs.Wrap(err, "Unable to get commit time")
		}

		if atTime.After(latest) {
			return atTime, nil
		}
	}

	return ts.Time, nil
}
