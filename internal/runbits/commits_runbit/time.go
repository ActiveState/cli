package commits_runbit

import (
	"time"

	"github.com/ActiveState/cli/internal/captain"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/pkg/checkoutinfo"
	"github.com/ActiveState/cli/pkg/platform/authentication"
	"github.com/ActiveState/cli/pkg/platform/model"
	"github.com/ActiveState/cli/pkg/project"
)

// ExpandTime returns a timestamp based on the given "--ts" value.
// If the timestamp was already defined, we just return it.
// If "now" was given, returns "now" according to the platform.
// Otherwise, returns the specified timestamp or nil (which falls back on the default Platform
// timestamp for a given operation)
func ExpandTime(ts *captain.TimeValue, auth *authentication.Auth) (time.Time, error) {
	if ts.Time != nil {
		return *ts.Time, nil
	}

	if ts.Now() {
		latest, err := model.FetchLatestRevisionTimeStamp(auth)
		if err != nil {
			return time.Time{}, errs.Wrap(err, "Unable to determine latest revision time")
		}
		return latest, nil
	}

	latest, err := model.FetchLatestTimeStamp(auth)
	if err != nil {
		return time.Time{}, errs.Wrap(err, "Unable to fetch latest Platform timestamp")
	}

	return latest, nil
}

// ExpandTimeForProject is the same as ExpandTime except that it ensures the returned time is either the same or
// later than that of the most recent commit.
func ExpandTimeForProject(ts *captain.TimeValue, auth *authentication.Auth, proj *project.Project) (time.Time, error) {
	timestamp, err := ExpandTime(ts, auth)
	if err != nil {
		return time.Time{}, errs.Wrap(err, "Unable to expand time")
	}

	if proj != nil {
		commitID, err := checkoutinfo.GetCommitID(proj.Dir())
		if err != nil {
			return time.Time{}, errs.Wrap(err, "Unable to get commit ID")
		}

		atTime, err := model.FetchTimeStampForCommit(commitID, auth)
		if err != nil {
			return time.Time{}, errs.Wrap(err, "Unable to get commit time")
		}

		if atTime.After(timestamp) {
			return *atTime, nil
		}
	}

	return timestamp, nil
}
