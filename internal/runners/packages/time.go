package packages

import (
	"time"

	"github.com/ActiveState/cli/internal/captain"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/pkg/localcommit"
	"github.com/ActiveState/cli/pkg/platform/authentication"
	"github.com/ActiveState/cli/pkg/platform/model"
	"github.com/ActiveState/cli/pkg/project"
)

// getTime returns a timestamp based on the given "--ts" value.
// If "now" was given, returns "now" according to the platform.
// If no timestamp was given but the current project's commit time is after the latest inventory
// timestamp, returns that commit time.
// Otherwise, returns the specified timestamp or nil (which falls back on the default Platform
// timestamp for a given operation)
func getTime(ts *captain.TimeValue, auth *authentication.Auth, proj *project.Project) (*time.Time, error) {
	if ts.Now() {
		latest, err := model.FetchLatestRevisionTimeStamp(auth)
		if err != nil {
			return nil, errs.Wrap(err, "Unable to determine latest revision time")
		}
		return &latest, nil
	}

	if ts.Time == nil && proj != nil {
		latest, err := model.FetchLatestTimeStamp(auth)
		if err != nil {
			return nil, errs.Wrap(err, "Unable to fetch latest Platform timestamp")
		}

		commitID, err := localcommit.Get(proj.Dir())
		if err != nil {
			return nil, errs.Wrap(err, "Unable to get commit ID")
		}

		atTime, err := model.FetchTimeStampForCommit(commitID, auth)
		if err != nil {
			return nil, errs.Wrap(err, "Unable to get commit time")
		}

		if atTime.After(latest) {
			return atTime, nil
		}
	}

	return ts.Time, nil
}
