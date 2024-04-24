package manifest

import (
	"errors"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/runbits/rationalize"
	"github.com/ActiveState/cli/pkg/platform/runtime/store"
)

func rationalizeError(rerr *error) {
	switch {
	case rerr == nil:
		return

	// No activestate.yaml.
	case errors.Is(*rerr, rationalize.ErrNoProject):
		*rerr = errs.WrapUserFacing(*rerr,
			locale.T("err_no_project"),
			errs.SetInput(),
		)
	case errs.Matches(*rerr, store.ErrNoBuildPlanFile):
		*rerr = errs.WrapUserFacing(*rerr,
			locale.Tl(
				"err_manifest_no_build_plan_file",
				"Could not source runtime. Please ensure your runtime is up to date by running '[ACTIONABLE]state refresh[/RESET]'.",
			),
			errs.SetInput(),
		)
	}
}
