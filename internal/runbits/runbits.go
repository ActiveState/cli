// Package runbits comprises logic that is shared between controllers, ie., code that prints
package runbits

import (
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/pkg/platform/api/buildplanner/response"
	"github.com/ActiveState/cli/pkg/platform/runtime/setup"
	"github.com/ActiveState/cli/pkg/platform/runtime/setup/buildlog"
)

func IsBuildError(err error) bool {
	return errs.Matches(err, &setup.BuildError{}) ||
		errs.Matches(err, &buildlog.BuildError{}) ||
		errs.Matches(err, &response.BuildPlannerError{})
}