package commit

import (
	"errors"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/primer"
	buildscript_runbit "github.com/ActiveState/cli/internal/runbits/buildscript"
	"github.com/ActiveState/cli/internal/runbits/cves"
	"github.com/ActiveState/cli/internal/runbits/dependencies"
	"github.com/ActiveState/cli/internal/runbits/rationalize"
	"github.com/ActiveState/cli/pkg/localcommit"
	bpResp "github.com/ActiveState/cli/pkg/platform/api/buildplanner/response"
	"github.com/ActiveState/cli/pkg/platform/model/buildplanner"
)

type primeable interface {
	primer.Outputer
	primer.Projecter
	primer.Auther
	primer.Analyticer
	primer.SvcModeler
	primer.Configurer
	primer.Prompter
}

type Commit struct {
	prime primeable
}

func New(p primeable) *Commit {
	return &Commit{p}
}

func rationalizeError(err *error) {
	var buildPlannerErr *bpResp.BuildPlannerError

	switch {
	case err == nil:
		return

	case errors.Is(*err, buildscript_runbit.ErrBuildscriptNotExist):
		*err = errs.WrapUserFacing(*err, locale.T("err_buildscript_not_exist"))

	// We communicate buildplanner errors verbatim as the intend is that these are curated by the buildplanner
	case errors.As(*err, &buildPlannerErr):
		*err = errs.WrapUserFacing(*err,
			buildPlannerErr.LocaleError(),
			errs.SetIf(buildPlannerErr.InputError(), errs.SetInput()))

	case errors.As(*err, &invalidDepsValueType{}):
		*err = errs.WrapUserFacing(*err, locale.T("err_commit_invalid_deps_value_type"), errs.SetInput())

	case errors.As(*err, &invalidDepValueType{}):
		*err = errs.WrapUserFacing(*err, locale.T("err_commit_invalid_dep_value_type"), errs.SetInput())

	}
}

func (c *Commit) Run() (rerr error) {
	defer rationalizeError(&rerr)

	proj := c.prime.Project()
	if proj == nil {
		return rationalize.ErrNoProject
	}

	out := c.prime.Output()
	out.Notice(locale.Tr("operating_message", proj.NamespaceString(), proj.Dir()))

	// Get buildscript.as representation
	script, err := buildscript_runbit.ScriptFromProject(proj)
	if err != nil {
		return errs.Wrap(err, "Could not get local build script")
	}

	for _, fc := range script.FunctionCalls("ingredient") {
		if err := NewIngredientCall(c.prime, script, fc).Resolve(); err != nil {
			return errs.Wrap(err, "Could not resolve ingredient")
		}
	}

	// Get equivalent build script for current state of the project
	localCommitID, err := localcommit.Get(proj.Dir())
	if err != nil {
		return errs.Wrap(err, "Unable to get local commit ID")
	}
	bp := buildplanner.NewBuildPlannerModel(c.prime.Auth(), c.prime.SvcModel())
	remoteScript, err := bp.GetBuildScript(localCommitID.String())
	if err != nil {
		return errs.Wrap(err, "Could not get remote build expr and time for provided commit")
	}

	equals, err := script.Equals(remoteScript)
	if err != nil {
		return errs.Wrap(err, "Could not compare local and remote build script")
	}

	// Check if there is anything to commit
	if equals {
		out.Notice(locale.Tl("commit_notice_no_change", "Your buildscript contains no new changes. No commit necessary."))
		return nil
	}

	pg := output.StartSpinner(out, locale.T("progress_commit"), constants.TerminalAnimationInterval)
	defer func() {
		if pg != nil {
			pg.Stop(locale.T("progress_fail"))
		}
	}()

	stagedCommit, err := bp.StageCommit(buildplanner.StageCommitParams{
		Owner:        proj.Owner(),
		Project:      proj.Name(),
		ParentCommit: localCommitID.String(),
		Script:       script,
	})
	if err != nil {
		return errs.Wrap(err, "Could not update project to reflect build script changes.")
	}

	// Update local commit ID
	if err := localcommit.Set(proj.Dir(), stagedCommit.CommitID.String()); err != nil {
		return errs.Wrap(err, "Could not set local commit ID")
	}

	// Update our local build expression to match the committed one. This allows our API a way to ensure forward compatibility.
	newScript, err := bp.GetBuildScript(stagedCommit.CommitID.String())
	if err != nil {
		return errs.Wrap(err, "Unable to get the remote build script")
	}
	if err := buildscript_runbit.Update(proj, newScript); err != nil {
		return errs.Wrap(err, "Could not update local build script")
	}

	pg.Stop(locale.T("progress_success"))
	pg = nil

	pgSolve := output.StartSpinner(out, locale.T("progress_solve"), constants.TerminalAnimationInterval)
	defer func() {
		if pgSolve != nil {
			pgSolve.Stop(locale.T("progress_fail"))
		}
	}()

	// Solve runtime
	rtCommit, err := bp.FetchCommit(stagedCommit.CommitID, proj.Owner(), proj.Name(), nil)
	if err != nil {
		return errs.Wrap(err, "Could not fetch staged commit")
	}

	// Get old buildplan.
	oldCommit, err := bp.FetchCommitNoPoll(localCommitID, proj.Owner(), proj.Name(), nil)
	if err != nil {
		return errs.Wrap(err, "Failed to fetch old commit")
	}

	pgSolve.Stop(locale.T("progress_success"))
	pgSolve = nil

	// Output dependency list.
	dependencies.OutputChangeSummary(out, rtCommit.BuildPlan(), oldCommit.BuildPlan())

	// Report CVEs.
	if err := cves.NewCveReport(c.prime).Report(rtCommit.BuildPlan(), oldCommit.BuildPlan()); err != nil {
		return errs.Wrap(err, "Could not report CVEs")
	}

	out.Notice("") // blank line
	out.Print(output.Prepare(
		locale.Tl(
			"commit_success",
			"", stagedCommit.CommitID.String(), proj.NamespaceString(),
		),
		&struct {
			Namespace string `json:"namespace"`
			Path      string `json:"path"`
			CommitID  string `json:"commit_id"`
		}{
			proj.NamespaceString(),
			proj.Dir(),
			stagedCommit.CommitID.String(),
		},
	))

	return nil
}
