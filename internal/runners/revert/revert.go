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
	"github.com/ActiveState/cli/internal/rtutils/ptr"
	"github.com/ActiveState/cli/internal/runbits/commit"
	"github.com/ActiveState/cli/internal/runbits/rationalize"
	runtime_runbit "github.com/ActiveState/cli/internal/runbits/runtime"
	"github.com/ActiveState/cli/internal/runbits/runtime/trigger"
	"github.com/ActiveState/cli/pkg/localcommit"
	gqlmodel "github.com/ActiveState/cli/pkg/platform/api/graphql/model"
	"github.com/ActiveState/cli/pkg/platform/authentication"
	"github.com/ActiveState/cli/pkg/platform/model"
	"github.com/ActiveState/cli/pkg/platform/model/buildplanner"
	"github.com/ActiveState/cli/pkg/project"
	"github.com/go-openapi/strfmt"
)

type Revert struct {
	prime primeable
	// The remainder is redundant with the above. Refactoring this will follow in a later story so as not to blow
	// up the one that necessitates adding the primer at this level.
	// https://activestatef.atlassian.net/browse/DX-2869
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
		prime,
		prime.Output(),
		prime.Prompt(),
		prime.Project(),
		prime.Auth(),
		prime.Analytics(),
		prime.SvcModel(),
		prime.Config(),
	}
}

const remoteCommitID = "REMOTE"
const headCommitID = "HEAD"

func (r *Revert) Run(params *Params) (rerr error) {
	defer rationalizeError(&rerr)

	if r.project == nil {
		return rationalize.ErrNoProject
	}

	commitID := params.CommitID
	if strings.EqualFold(commitID, headCommitID) {
		r.out.Notice(locale.T("warn_revert_head"))
		commitID = remoteCommitID
	}
	if !strfmt.IsUUID(commitID) && !strings.EqualFold(commitID, remoteCommitID) {
		return locale.NewInputError("err_revert_invalid_commit_id", "Invalid commit ID")
	}
	latestCommit, err := localcommit.Get(r.project.Dir())
	if err != nil {
		return errs.Wrap(err, "Unable to get local commit")
	}
	if strings.EqualFold(commitID, remoteCommitID) {
		commitID = latestCommit.String()
	}

	if commitID == latestCommit.String() && params.To {
		return locale.NewInputError("err_revert_to_current_commit", "The commit to revert to cannot be the latest commit")
	}
	r.out.Notice(locale.Tr("operating_message", r.project.NamespaceString(), r.project.Dir()))

	bp := buildplanner.NewBuildPlannerModel(r.auth, r.prime.SvcModel())
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

	targetCommit, err := model.GetCommitWithinCommitHistory(latestCommit, strfmt.UUID(targetCommitID), r.auth)
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
		orgs, err = model.FetchOrganizationsByIDs([]strfmt.UUID{*targetCommit.Author}, r.auth)
		if err != nil {
			return locale.WrapError(err, "err_revert_get_organizations", "Could not get organizations for current user")
		}
	}

	if !r.out.Type().IsStructured() {
		r.out.Print(locale.Tl("revert_info", "You are about to revert{{.V0}} the following commit:", preposition))
		if err := commit.PrintCommit(r.out, targetCommit, orgs); err != nil {
			return locale.WrapError(err, "err_revert_print_commit", "Could not print commit")
		}
	}

	defaultChoice := !r.prime.Prompt().IsInteractive()
	revert, err := r.prime.Prompt().Confirm("", locale.Tl("revert_confirm", "Continue?"), &defaultChoice, ptr.To(true))
	if err != nil {
		return errs.Wrap(err, "Not confirmed")
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

	_, err = runtime_runbit.Update(r.prime, trigger.TriggerRevert)
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

type revertParams struct {
	organization   string
	project        string
	parentCommitID string
	revertCommitID string
}

func (r *Revert) revertCommit(params revertParams, bp *buildplanner.BuildPlanner) (strfmt.UUID, error) {
	newCommitID, err := bp.RevertCommit(params.organization, params.project, params.parentCommitID, params.revertCommitID)
	if err != nil {
		return "", errs.Wrap(err, "Could not revert commit")
	}

	return newCommitID, nil
}

func (r *Revert) revertToCommit(params revertParams, bp *buildplanner.BuildPlanner) (strfmt.UUID, error) {
	bs, err := bp.GetBuildScript(params.revertCommitID)
	if err != nil {
		return "", errs.Wrap(err, "Could not get build expression")
	}

	stageCommitParams := buildplanner.StageCommitParams{
		Owner:        params.organization,
		Project:      params.project,
		ParentCommit: params.parentCommitID,
		Description:  locale.Tl("revert_commit_description", "Revert to commit {{.V0}}", params.revertCommitID),
		Script:       bs,
	}

	newCommit, err := bp.StageCommit(stageCommitParams)
	if err != nil {
		return "", errs.Wrap(err, "Could not stage commit")
	}

	return newCommit.CommitID, nil
}
