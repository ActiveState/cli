package commit

import (
	"errors"

	"github.com/ActiveState/cli/internal/analytics"
	"github.com/ActiveState/cli/internal/config"
	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/primer"
	"github.com/ActiveState/cli/internal/runbits/buildscript"
	"github.com/ActiveState/cli/internal/runbits/rationalize"
	"github.com/ActiveState/cli/pkg/localcommit"
	bpResp "github.com/ActiveState/cli/pkg/platform/api/buildplanner/response"
	"github.com/ActiveState/cli/pkg/platform/authentication"
	"github.com/ActiveState/cli/pkg/platform/model"
	"github.com/ActiveState/cli/pkg/platform/model/buildplanner"
	"github.com/ActiveState/cli/pkg/project"
)

type primeable interface {
	primer.Outputer
	primer.Projecter
	primer.Auther
	primer.Analyticer
	primer.SvcModeler
	primer.Configurer
}

type Commit struct {
	out       output.Outputer
	proj      *project.Project
	auth      *authentication.Auth
	analytics analytics.Dispatcher
	svcModel  *model.SvcModel
	cfg       *config.Instance
}

func New(p primeable) *Commit {
	return &Commit{
		out:       p.Output(),
		proj:      p.Project(),
		auth:      p.Auth(),
		analytics: p.Analytics(),
		svcModel:  p.SvcModel(),
		cfg:       p.Config(),
	}
}

var ErrNoChanges = errors.New("buildscript has no changes")

func rationalizeError(err *error) {
	var buildPlannerErr *bpResp.BuildPlannerError

	switch {
	case err == nil:
		return

	case errors.Is(*err, ErrNoChanges):
		*err = errs.WrapUserFacing(*err, locale.Tl(
			"commit_notice_no_change",
			"No change to the buildscript was found.",
		), errs.SetInput())

	case errs.Matches(*err, buildscript_runbit.ErrBuildscriptNotExist):
		*err = errs.WrapUserFacing(*err, locale.T("err_buildscript_not_exist"))

	// We communicate buildplanner errors verbatim as the intend is that these are curated by the buildplanner
	case errors.As(*err, &buildPlannerErr):
		*err = errs.WrapUserFacing(*err,
			buildPlannerErr.LocalizedError(),
			errs.SetIf(buildPlannerErr.InputError(), errs.SetInput()))
	}
}

func (c *Commit) Run() (rerr error) {
	defer rationalizeError(&rerr)

	if c.proj == nil {
		return rationalize.ErrNoProject
	}

	pg := output.StartSpinner(c.out, locale.T("progress_commit"), constants.TerminalAnimationInterval)
	defer func() {
		if pg != nil {
			pg.Stop(locale.T("progress_fail") + "\n")
		}
	}()

	// Get buildscript.as representation
	script, err := buildscript_runbit.ScriptFromProject(c.proj)
	if err != nil {
		return errs.Wrap(err, "Could not get local build script")
	}

	// Get equivalent build script for current state of the project
	localCommitID, err := localcommit.Get(c.proj.Dir())
	if err != nil {
		return errs.Wrap(err, "Unable to get local commit ID")
	}
	bp := buildplanner.NewBuildPlannerModel(c.auth)
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
		return ErrNoChanges
	}

	stagedCommitID, err := bp.StageCommit(buildplanner.StageCommitParams{
		Owner:        c.proj.Owner(),
		Project:      c.proj.Name(),
		ParentCommit: localCommitID.String(),
		Script:       script,
	})
	if err != nil {
		return errs.Wrap(err, "Could not update project to reflect build script changes.")
	}

	// Update local commit ID
	if err := localcommit.Set(c.proj.Dir(), stagedCommitID.String()); err != nil {
		return errs.Wrap(err, "Could not set local commit ID")
	}

	// Update our local build expression to match the committed one. This allows our API a way to ensure forward compatibility.
	newScript, err := bp.GetBuildScript(stagedCommitID.String())
	if err != nil {
		return errs.Wrap(err, "Unable to get the remote build script")
	}
	if err := buildscript_runbit.Update(c.proj, newScript); err != nil {
		return errs.Wrap(err, "Could not update local build script")
	}

	pg.Stop(locale.T("progress_success") + "\n")
	pg = nil

	c.out.Print(output.Prepare(
		locale.Tl(
			"commit_success",
			"", stagedCommitID.String(), c.proj.NamespaceString(),
		),
		&struct {
			Namespace string `json:"namespace"`
			Path      string `json:"path"`
			CommitID  string `json:"commit_id"`
		}{
			c.proj.NamespaceString(),
			c.proj.Dir(),
			stagedCommitID.String(),
		},
	))

	return nil
}
