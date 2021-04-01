package branch

import (
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/runbits"
	"github.com/ActiveState/cli/pkg/platform/model"
	"github.com/ActiveState/cli/pkg/project"
)

type configurable interface {
	CachePath() string
}

type Switch struct {
	out     output.Outputer
	project *project.Project
	config  configurable
}

type SwitchParams struct {
	Name string
}

func NewSwitch(prime primeable) *Switch {
	return &Switch{
		out:     prime.Output(),
		project: prime.Project(),
		config:  prime.Config(),
	}
}

func (s *Switch) Run(params SwitchParams) error {
	logging.Debug("ExecuteSwitch")

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

	err = s.project.Source().SetCommit(branch.CommitID.String(), s.project.IsHeadless())
	if err != nil {
		return locale.WrapError(err, "err_switch_set_commitID", "Could not update commit ID")
	}

	err = runbits.RefreshRuntime(s.out, s.project, s.config.CachePath(), *branch.CommitID, false)
	if err != nil {
		return locale.WrapError(err, "err_refresh_runtime")
	}

	s.out.Print(locale.Tl("branch_switch_success", "Successfully switched to branch: [NOTICE]{{.V0}}[/RESET]", params.Name))

	return nil
}
