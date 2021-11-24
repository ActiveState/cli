package branch

import (
	"github.com/ActiveState/cli/internal/analytics"
	"github.com/ActiveState/cli/internal/installation/storage"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/runbits"
	"github.com/ActiveState/cli/pkg/platform/authentication"
	"github.com/ActiveState/cli/pkg/platform/model"
	"github.com/ActiveState/cli/pkg/platform/runtime/target"
	"github.com/ActiveState/cli/pkg/project"
)

type Switch struct {
	auth      *authentication.Auth
	out       output.Outputer
	project   *project.Project
	analytics analytics.Dispatcher
	svcModel  *model.SvcModel
}

type SwitchParams struct {
	Name string
}

func NewSwitch(prime primeable) *Switch {
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

	branch, err := model.BranchForProjectByName(project, params.Name)
	if err != nil {
		return locale.WrapError(err, "err_fetch_branch", "", params.Name)
	}

	err = s.project.Source().SetBranch(branch.Label)
	if err != nil {
		return locale.WrapError(err, "err_switch_set_branch", "Could not update branch")
	}

	err = s.project.SetCommit(branch.CommitID.String())
	if err != nil {
		return locale.WrapError(err, "err_switch_set_commitID", "Could not update commit ID")
	}

	err = runbits.RefreshRuntime(s.auth, s.out, s.analytics, s.project, storage.CachePath(), *branch.CommitID, false, target.TriggerBranch, s.svcModel)
	if err != nil {
		return locale.WrapError(err, "err_refresh_runtime")
	}

	s.out.Print(locale.Tl("branch_switch_success", "Successfully switched to branch: [NOTICE]{{.V0}}[/RESET]", params.Name))

	return nil
}
