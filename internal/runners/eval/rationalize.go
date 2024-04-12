package eval

import (
	"errors"
	"strings"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/runbits/rationalize"
	"github.com/ActiveState/cli/pkg/platform/api/buildplanner/model"
	bpResp "github.com/ActiveState/cli/pkg/platform/api/buildplanner/response"
)

func rationalizeError(rerr *error) {
	if rerr == nil {
		return
	}

	var planningError *bpResp.BuildPlannerError
	var failedArtifactsError model.ErrFailedArtifacts

	switch {
	case errors.Is(*rerr, rationalize.ErrNotAuthenticated):
		*rerr = errs.WrapUserFacing(*rerr,
			locale.T("err_init_authenticated"),
			errs.SetInput(),
		)

	case errors.Is(*rerr, rationalize.ErrNoProject):
		*rerr = errs.WrapUserFacing(*rerr,
			locale.Tr("err_no_project"),
			errs.SetInput())

	case errors.As(*rerr, &planningError):
		// Forward API error to user.
		*rerr = errs.WrapUserFacing(*rerr, planningError.Error())

	case errors.As(*rerr, &failedArtifactsError):
		var artfErrs []string
		for _, artf := range failedArtifactsError.Artifacts {
			artfErrs = append(artfErrs, locale.Tr("err_build_artifact_failed", artf.DisplayName, strings.Join(artf.Errors, "\n"), artf.LogURL))
		}
		*rerr = errs.WrapUserFacing(*rerr, locale.Tr("err_build_artifacts_failed", strings.Join(artfErrs, "\n")))
	}
}
