package swtch

import (
	"errors"

	"github.com/ActiveState/cli/internal/analytics"
	"github.com/ActiveState/cli/internal/config"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/primer"
	"github.com/ActiveState/cli/internal/runbits/rationalize"
	"github.com/ActiveState/cli/internal/runbits/runtime"
	"github.com/ActiveState/cli/internal/runbits/runtime/trigger"
	"github.com/ActiveState/cli/pkg/localcommit"
	"github.com/ActiveState/cli/pkg/platform/api/mono/mono_models"
	"github.com/ActiveState/cli/pkg/platform/authentication"
	"github.com/ActiveState/cli/pkg/platform/model"
	"github.com/ActiveState/cli/pkg/project"
	"github.com/go-openapi/strfmt"
)

type Switch struct {
	prime primeable
	// The remainder is redundant with the above. Refactoring this will follow in a later story so as not to blow
	// up the one that necessitates adding the primer at this level.
	// https://activestatef.atlassian.net/browse/DX-2869
	auth      *authentication.Auth
	out       output.Outputer
	project   *project.Project
	analytics analytics.Dispatcher
	svcModel  *model.SvcModel
	cfg       *config.Instance
}

type SwitchParams struct {
	Identifier string
}

type primeable interface {
	primer.Auther
	primer.Outputer
	primer.Projecter
	primer.Configurer
	primer.Analyticer
	primer.SvcModeler
}

type identifier interface {
	CommitID() strfmt.UUID
	Locale() string
}

type commitIdentifier struct {
	commitID strfmt.UUID
}

func (c commitIdentifier) CommitID() strfmt.UUID {
	return c.commitID
}

func (c commitIdentifier) Locale() string {
	return locale.Tl("commit_identifier_type", "commit")
}

type branchIdentifier struct {
	branch *mono_models.Branch
}

func (b branchIdentifier) CommitID() strfmt.UUID {
	return *b.branch.CommitID
}

func (b branchIdentifier) Locale() string {
	return locale.Tl("branch_identifier_type", "branch")
}

type errCommitNotOnBranch struct {
	commitID string
	branch   string
}

func (e errCommitNotOnBranch) Error() string {
	return "commit is not on branch"
}

func New(prime primeable) *Switch {
	return &Switch{
		prime:     prime,
		auth:      prime.Auth(),
		out:       prime.Output(),
		project:   prime.Project(),
		analytics: prime.Analytics(),
		svcModel:  prime.SvcModel(),
		cfg:       prime.Config(),
	}
}

func rationalizeError(rerr *error) {
	if rerr == nil {
		return
	}

	var commitNotOnBranchErr *errCommitNotOnBranch

	switch {
	case errors.As(*rerr, &commitNotOnBranchErr):
		*rerr = errs.WrapUserFacing(*rerr,
			locale.Tl("err_identifier_branch_not_on_branch", "Commit does not belong to history for branch [ACTIONABLE]{{.V0}}[/RESET]", commitNotOnBranchErr.branch),
			errs.SetInput(),
		)
	}
}

func (s *Switch) Run(params SwitchParams) (rerr error) {
	defer rationalizeError(&rerr)
	logging.Debug("ExecuteSwitch")

	if s.project == nil {
		return rationalize.ErrNoProject
	}
	s.out.Notice(locale.Tr("operating_message", s.project.NamespaceString(), s.project.Dir()))

	project, err := model.LegacyFetchProjectByName(s.project.Owner(), s.project.Name())
	if err != nil {
		return errs.Wrap(err, "Could not fetch project '%s'", s.project.Namespace().String())
	}

	identifier, err := resolveIdentifier(project, params.Identifier)
	if err != nil {
		return errs.Wrap(err, "Could not resolve identifier '%s'", params.Identifier)
	}

	if id, ok := identifier.(branchIdentifier); ok {
		err = s.project.Source().SetBranch(id.branch.Label)
		if err != nil {
			return errs.Wrap(err, "Could not update branch")
		}
	}

	belongs, err := model.CommitBelongsToBranch(s.project.Owner(), s.project.Name(), s.project.BranchName(), identifier.CommitID(), s.auth)
	if err != nil {
		return errs.Wrap(err, "Could not determine if commit belongs to branch")
	}
	if !belongs {
		return &errCommitNotOnBranch{identifier.CommitID().String(), s.project.BranchName()}
	}

	err = localcommit.Set(s.project.Dir(), identifier.CommitID().String())
	if err != nil {
		return errs.Wrap(err, "Unable to set local commit")
	}

	_, err = runtime_runbit.Update(s.prime, trigger.TriggerSwitch)
	if err != nil {
		return errs.Wrap(err, "Could not setup runtime")
	}

	s.out.Print(output.Prepare(
		locale.Tl("branch_switch_success", "Successfully switched to {{.V0}}: [NOTICE]{{.V1}}[/RESET]", identifier.Locale(), params.Identifier),
		&struct {
			Branch string `json:"branch"`
		}{
			params.Identifier,
		},
	))

	return nil
}

func resolveIdentifier(project *mono_models.Project, idParam string) (identifier, error) {
	if strfmt.IsUUID(idParam) {
		return commitIdentifier{strfmt.UUID(idParam)}, nil
	}

	branch, err := model.BranchForProjectByName(project, idParam)
	if err != nil {
		return nil, errs.Wrap(err, "Could not get branch '%s'", idParam)

	}

	return branchIdentifier{branch: branch}, nil
}
