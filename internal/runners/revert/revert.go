package revert

import (
	"strings"

	"github.com/ActiveState/cli/internal/analytics"
	"github.com/ActiveState/cli/internal/config"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/primer"
	"github.com/ActiveState/cli/internal/prompt"
	"github.com/ActiveState/cli/internal/runbits"
	"github.com/ActiveState/cli/internal/runbits/commit"
	"github.com/ActiveState/cli/pkg/localcommit"
	gqlmodel "github.com/ActiveState/cli/pkg/platform/api/graphql/model"
	"github.com/ActiveState/cli/pkg/platform/authentication"
	"github.com/ActiveState/cli/pkg/platform/model"
	"github.com/ActiveState/cli/pkg/platform/runtime/target"
	"github.com/ActiveState/cli/pkg/project"
	"github.com/go-openapi/strfmt"
)

type Revert struct {
	out       output.Outputer
	prompt    prompt.Prompter
	project   *project.Project
	auth      *authentication.Auth
	analytics analytics.Dispatcher
	svcModel  *model.SvcModel
	cfg       *config.Instance
}

type Params struct {
	CommitID string
	To       bool
	Force    bool
}

type primeable interface {
	primer.Outputer
	primer.Prompter
	primer.Projecter
	primer.Auther
	primer.Analyticer
	primer.SvcModeler
	primer.Configurer
}

func New(prime primeable) *Revert {
	return &Revert{
		prime.Output(),
		prime.Prompt(),
		prime.Project(),
		prime.Auth(),
		prime.Analytics(),
		prime.SvcModel(),
		prime.Config(),
	}
}

const headCommitID = "HEAD"

func (r *Revert) Run(params *Params) (rerr error) {
	defer rationalizeError(&rerr)

	if r.project == nil {
		return locale.NewInputError("err_no_project")
	}

	commitID := params.CommitID
	if !strfmt.IsUUID(commitID) && !strings.EqualFold(commitID, headCommitID) {
		return locale.NewInputError("err_invalid_commit_id", "Invalid commit ID")
	}
	latestCommit, err := localcommit.Get(r.project.Dir())
	if err != nil {
		return errs.Wrap(err, "Unable to get local commit")
	}
	if strings.EqualFold(commitID, headCommitID) {
		commitID = latestCommit.String()
	}

	if commitID == latestCommit.String() && params.To {
		return locale.NewInputError("err_revert_to_current_commit", "The commit to revert to cannot be the latest commit")
	}
	r.out.Notice(locale.Tr("operating_message", r.project.NamespaceString(), r.project.Dir()))

	bp := model.NewBuildPlannerModel(r.auth)
	targetCommitID := commitID // the commit to revert the contents of, or the commit to revert to
	revertParams := revertParams{
		organization:   r.project.Owner(),
		project:        r.project.Name(),
		parentCommitID: latestCommit.String(),
		revertCommitID: commitID,
	}
	revertFunc := r.revertCommit
	preposition := ""
	if params.To {
		revertFunc = r.revertToCommit
		preposition = " to" // need leading whitespace
	}

	targetCommit, err := model.GetCommitWithinCommitHistory(latestCommit, strfmt.UUID(targetCommitID))
	if err != nil {
		if err == model.ErrCommitNotInHistory {
			return locale.WrapInputError(err, "err_revert_commit_not_found", "The commit [NOTICE]{{.V0}}[/RESET] was not found in the project's commit history.", commitID)
		}
		return errs.AddTips(
			locale.WrapError(err, "err_revert_get_commit", "", commitID),
			locale.T("tip_private_project_auth"),
		)
	}

	var orgs []gqlmodel.Organization
	if targetCommit.Author != nil {
		var err error
		orgs, err = model.FetchOrganizationsByIDs([]strfmt.UUID{*targetCommit.Author})
		if err != nil {
			return locale.WrapError(err, "err_revert_get_organizations", "Could not get organizations for current user")
		}
	}

	if !r.out.Type().IsStructured() {
		r.out.Print(locale.Tl("revert_info", "You are about to revert{{.V0}} the following commit:", preposition))
		commit.PrintCommit(r.out, targetCommit, orgs)
	}

	defaultChoice := params.Force || !r.out.Config().Interactive
	revert, err := r.prompt.Confirm("", locale.Tl("revert_confirm", "Continue?"), &defaultChoice)
	if err != nil {
		return locale.WrapError(err, "err_revert_confirm", "Could not confirm revert choice")
	}
	if !revert {
		return locale.NewInputError("err_revert_aborted", "Revert aborted by user")
	}

	revertCommit, err := revertFunc(revertParams, bp)
	if err != nil {
		return errs.AddTips(
			locale.WrapError(err, "err_revert_commit", "", preposition, commitID),
			locale.Tl("tip_revert_sync", "Please ensure that the local project is synchronized with the platform and that the given commit ID belongs to the current project"),
			locale.T("tip_private_project_auth"))
	}

	err = localcommit.Set(r.project.Dir(), revertCommit.String())
	if err != nil {
		return errs.Wrap(err, "Unable to set local commit")
	}

	err = runbits.RefreshRuntime(r.auth, r.out, r.analytics, r.project, revertCommit, true, target.TriggerRevert, r.svcModel, r.cfg)
	if err != nil {
		return locale.WrapError(err, "err_refresh_runtime")
	}

	r.out.Print(output.Prepare(
		locale.Tl("revert_success", "Successfully reverted{{.V0}} commit: {{.V1}}", preposition, commitID),
		&struct {
			CurrentCommitID string `json:"current_commit_id"`
		}{
			revertCommit.String(),
		},
	))
	r.out.Notice(locale.T("operation_success_local"))
	return nil
}

type revertFunc func(params revertParams, bp *model.BuildPlanner) (strfmt.UUID, error)

type revertParams struct {
	organization   string
	project        string
	parentCommitID string
	revertCommitID string
}

func (r *Revert) revertCommit(params revertParams, bp *model.BuildPlanner) (strfmt.UUID, error) {
	newCommitID, err := bp.RevertCommit(params.organization, params.project, params.parentCommitID, params.revertCommitID)
	if err != nil {
		return "", errs.Wrap(err, "Could not revert commit")
	}

	return newCommitID, nil
}

func (r *Revert) revertToCommit(params revertParams, bp *model.BuildPlanner) (strfmt.UUID, error) {
	buildExpression, err := bp.GetBuildExpression(params.organization, params.project, params.revertCommitID)
	if err != nil {
		return "", errs.Wrap(err, "Could not get build expression")
	}

	stageCommitParams := model.StageCommitParams{
		Owner:        params.organization,
		Project:      params.project,
		ParentCommit: params.parentCommitID,
		Description:  locale.Tl("revert_commit_description", "Revert to commit {{.V0}}", params.revertCommitID),
		Expression:   buildExpression,
	}

	newCommitID, err := bp.StageCommit(stageCommitParams)
	if err != nil {
		return "", errs.Wrap(err, "Could not stage commit")
	}

	return newCommitID, nil
}
