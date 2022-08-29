package swtch

import (
	"github.com/ActiveState/cli/internal/analytics"
	"github.com/ActiveState/cli/internal/installation/storage"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/primer"
	"github.com/ActiveState/cli/internal/runbits"
	"github.com/ActiveState/cli/pkg/platform/api/mono/mono_models"
	"github.com/ActiveState/cli/pkg/platform/authentication"
	"github.com/ActiveState/cli/pkg/platform/model"
	"github.com/ActiveState/cli/pkg/platform/runtime/target"
	"github.com/ActiveState/cli/pkg/project"
	"github.com/go-openapi/strfmt"
)

type Switch struct {
	auth      *authentication.Auth
	out       output.Outputer
	project   *project.Project
	analytics analytics.Dispatcher
	svcModel  *model.SvcModel
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

func New(prime primeable) *Switch {
	return &Switch{
		auth:      prime.Auth(),
		out:       prime.Output(),
		project:   prime.Project(),
		analytics: prime.Analytics(),
		svcModel:  prime.SvcModel(),
	}
}

func (s *Switch) Run(params SwitchParams) error {
	logging.Debug("ExecuteSwitch")

	if s.project == nil {
		return locale.NewInputError("err_no_project")
	}

	project, err := model.FetchProjectByName(s.project.Owner(), s.project.Name())
	if err != nil {
		return locale.WrapError(err, "err_fetch_project", "", s.project.Namespace().String())
	}

	identifier, err := resolveIdentifier(project, params.Identifier)
	if err != nil {
		return locale.WrapError(err, "err_resolve_identifier", "Could not resolve identifier {{.V0}}", params.Identifier)
	}

	if id, ok := identifier.(branchIdentifier); ok {
		err = s.project.Source().SetBranch(id.branch.Label)
		if err != nil {
			return locale.WrapError(err, "err_switch_set_branch", "Could not update branch")
		}
	}

	err = s.project.SetCommit(identifier.CommitID().String())
	if err != nil {
		return locale.WrapError(err, "err_switch_set_commitID", "Could not update commit ID")
	}

	err = runbits.RefreshRuntime(s.auth, s.out, s.analytics, s.project, storage.CachePath(), identifier.CommitID(), false, target.TriggerSwitch, s.svcModel)
	if err != nil {
		return locale.WrapError(err, "err_refresh_runtime")
	}

	s.out.Print(locale.Tl("branch_switch_success", "Successfully switched to {{.V0}}: [NOTICE]{{.V1}}[/RESET]", identifier.Locale(), params.Identifier))

	return nil
}

func resolveIdentifier(project *mono_models.Project, idParam string) (identifier, error) {
	if strfmt.IsUUID(idParam) {
		return commitIdentifier{strfmt.UUID(idParam)}, nil
	}

	branch, err := model.BranchForProjectByName(project, idParam)
	if err != nil {
		return nil, locale.WrapError(err, "err_identifier_branch", "Could not get branch {{.V0}} for current project", idParam)

	}

	return branchIdentifier{branch: branch}, nil
}
