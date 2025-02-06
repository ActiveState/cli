package platforms

import (
	"errors"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/primer"
	"github.com/ActiveState/cli/internal/rtutils/ptr"
	"github.com/ActiveState/cli/internal/runbits/commits_runbit"
	"github.com/ActiveState/cli/internal/runbits/rationalize"
	"github.com/ActiveState/cli/internal/runbits/rationalizers"
	"github.com/ActiveState/cli/internal/runbits/reqop_runbit"
	"github.com/ActiveState/cli/internal/runbits/runtime/trigger"
	"github.com/ActiveState/cli/pkg/localcommit"
	bpResp "github.com/ActiveState/cli/pkg/platform/api/buildplanner/response"
	"github.com/ActiveState/cli/pkg/platform/model"
	bpModel "github.com/ActiveState/cli/pkg/platform/model/buildplanner"
)

// AddRunParams tracks the info required for running Add.
type AddRunParams struct {
	Params
}

// Add manages the adding execution context.
type Add struct {
	prime primeable
}

type primeable interface {
	primer.Outputer
	primer.Prompter
	primer.Projecter
	primer.Auther
	primer.Configurer
	primer.Analyticer
	primer.SvcModeler
}

// NewAdd prepares an add execution context for use.
func NewAdd(prime primeable) *Add {
	return &Add{
		prime: prime,
	}
}

// Run executes the add behavior.
func (a *Add) Run(params AddRunParams) (rerr error) {
	defer rationalizeAddPlatformError(&rerr)

	logging.Debug("Execute platforms add")

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

	pg = output.StartSpinner(out, locale.T("progress_platform_search"), constants.TerminalAnimationInterval)

	// Grab local commit info
	localCommitID, err := localcommit.Get(pj.Dir())
	if err != nil {
		return errs.Wrap(err, "Unable to get local commit")
	}
	oldCommit, err := bp.FetchCommit(localCommitID, pj.Owner(), pj.Name(), nil)
	if err != nil {
		return errs.Wrap(err, "Failed to fetch old build result")
	}

	// Resolve platform
	platform, err := model.FetchPlatformByDetails(params.Platform.Name(), params.Platform.Version(), params.BitWidth)
	if err != nil {
		return errs.Wrap(err, "Could not fetch platform")
	}

	pg.Stop(locale.T("progress_found"))
	pg = nil

	// Resolve timestamp, commit and languages used for current project.
	ts, err := commits_runbit.ExpandTimeForProject(nil, a.prime.Auth(), pj)
	if err != nil {
		return errs.Wrap(err, "Unable to get timestamp from params")
	}

	// Prepare updated buildscript
	script := oldCommit.BuildScript()
	script.SetAtTime(ts, true)
	script.AddPlatform(*platform.PlatformID)

	// Update local checkout and source runtime changes
	if err := reqop_runbit.UpdateAndReload(a.prime, script, oldCommit, locale.Tr("commit_message_added", *platform.DisplayName), trigger.TriggerPlatform); err != nil {
		return errs.Wrap(err, "Failed to update local checkout")
	}

	out.Notice(locale.Tr("platform_added", *platform.DisplayName))

	if out.Type().IsStructured() {
		out.Print(output.Structured(platform))
	}

	return nil
}

func rationalizeAddPlatformError(rerr *error) {
	switch {
	case rerr == nil:
		return

	// No matches found
	case errors.Is(*rerr, model.ErrPlatformNotFound):
		*rerr = errs.WrapUserFacing(
			*rerr,
			locale.Tr("platform_add_not_found"),
			errs.SetInput(),
		)

	// Error staging a commit during install.
	case errors.As(*rerr, ptr.To(&bpResp.CommitError{})):
		rationalizers.HandleCommitErrors(rerr)

	}
}
