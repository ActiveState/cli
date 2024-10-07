package platforms

import (
	"errors"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/rtutils/ptr"
	"github.com/ActiveState/cli/internal/runbits/rationalize"
	"github.com/ActiveState/cli/internal/runbits/rationalizers"
	"github.com/ActiveState/cli/internal/runbits/reqop_runbit"
	"github.com/ActiveState/cli/internal/runbits/runtime/trigger"
	"github.com/ActiveState/cli/pkg/localcommit"
	bpResp "github.com/ActiveState/cli/pkg/platform/api/buildplanner/response"
	"github.com/ActiveState/cli/pkg/platform/model"
	bpModel "github.com/ActiveState/cli/pkg/platform/model/buildplanner"
)

type RemoveRunParams struct {
	Params
}

type Remove struct {
	prime primeable
}

func NewRemove(prime primeable) *Remove {
	return &Remove{
		prime: prime,
	}
}

var errNoMatch = errors.New("no platform matched the search criteria")
var errMultiMatch = errors.New("multiple platforms matched the search criteria")

func (a *Remove) Run(params RemoveRunParams) (rerr error) {
	defer rationalizeRemovePlatformError(&rerr)

	logging.Debug("Execute platforms remove")

	if a.prime.Project() == nil {
		return rationalize.ErrNoProject
	}

	pj := a.prime.Project()
	out := a.prime.Output()
	bp := bpModel.NewBuildPlannerModel(a.prime.Auth(), a.prime.SvcModel())

	var pg *output.Spinner
	defer func() {
		if pg != nil {
			pg.Stop(locale.T("progress_fail"))
		}
	}()

	pg = output.StartSpinner(out, locale.T("progress_platforms"), constants.TerminalAnimationInterval)

	// Grab local commit info
	localCommitID, err := localcommit.Get(pj.Dir())
	if err != nil {
		return errs.Wrap(err, "Unable to get local commit")
	}
	oldCommit, err := bp.FetchCommit(localCommitID, pj.Owner(), pj.Name(), nil)
	if err != nil {
		return errs.Wrap(err, "Failed to fetch old build result")
	}

	pg.Stop(locale.T("progress_found"))
	pg = nil

	// Prepare updated buildscript
	script := oldCommit.BuildScript()
	platforms, err := script.Platforms("")
	if err != nil {
		return errs.Wrap(err, "Failed to get platforms")
	}
	toRemove := []*model.Platform{}
	for _, uid := range platforms {
		platform, err := model.FetchPlatformByUID(uid)
		if err != nil {
			return errs.Wrap(err, "Failed to get platform")
		}
		if model.IsPlatformMatch(platform, params.Platform.Name(), params.Platform.Version(), params.BitWidth) {
			toRemove = append(toRemove, platform)
		}
	}
	if len(toRemove) == 0 {
		return errNoMatch
	}
	if len(toRemove) > 1 {
		return errMultiMatch
	}

	if err := script.RemovePlatform(*toRemove[0].PlatformID); err != nil {
		return errs.Wrap(err, "Failed to remove platform")
	}

	// Update local checkout and source runtime changes
	if err := reqop_runbit.UpdateAndReload(a.prime, script, oldCommit, locale.Tr("commit_message_added", params.Platform.String()), trigger.TriggerPlatform); err != nil {
		return errs.Wrap(err, "Failed to update local checkout")
	}

	out.Notice(locale.Tr("platform_added", params.Platform.String()))

	if out.Type().IsStructured() {
		out.Print(output.Structured(toRemove[0]))
	}

	return nil
}

func rationalizeRemovePlatformError(rerr *error) {
	switch {
	case rerr == nil:
		return

	// No matches found
	case errors.Is(*rerr, errNoMatch):
		*rerr = errs.WrapUserFacing(
			*rerr,
			locale.Tr("err_uninstall_platform_nomatch"),
			errs.SetInput(),
		)

	// Multiple matches found
	case errors.Is(*rerr, errMultiMatch):
		*rerr = errs.WrapUserFacing(
			*rerr,
			locale.Tr("err_uninstall_platform_multimatch"),
			errs.SetInput(),
		)

	// Error staging a commit during install.
	case errors.As(*rerr, ptr.To(&bpResp.CommitError{})):
		rationalizers.HandleCommitErrors(rerr)

	}
}
