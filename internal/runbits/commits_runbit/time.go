package commits_runbit

import (
	"time"

	"github.com/ActiveState/cli/internal/captain"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/pkg/platform/authentication"
	"github.com/ActiveState/cli/pkg/platform/model"
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
