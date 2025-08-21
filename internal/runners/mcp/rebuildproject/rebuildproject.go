package rebuildproject

import (
	"encoding/json"
	"fmt"

	"github.com/ActiveState/cli/internal/captain"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/primer"
	"github.com/ActiveState/cli/internal/runbits/commits_runbit"
	"github.com/ActiveState/cli/pkg/platform/authentication"
	"github.com/ActiveState/cli/pkg/platform/model"
	bpModel "github.com/ActiveState/cli/pkg/platform/model/buildplanner"
	"github.com/ActiveState/cli/pkg/project"
)

type ProjectErrorsRunner struct {
	auth     *authentication.Auth
	output   output.Outputer
	svcModel *model.SvcModel
}

func New(p *primer.Values) *ProjectErrorsRunner {
	return &ProjectErrorsRunner{
		auth:     p.Auth(),
		output:   p.Output(),
		svcModel: p.SvcModel(),
	}
}

type Params struct {
	project *project.Namespaced
}

func NewParams(project *project.Namespaced) *Params {
	return &Params{
		project: project,
	}
}

func (runner *ProjectErrorsRunner) Run(params *Params) error {
	branch, err := model.DefaultBranchForProjectName(params.project.Owner, params.project.Project)
	if err != nil {
		return fmt.Errorf("error fetching default branch: %w", err)
	}

	// Collect "before" buildplan
	bpm := bpModel.NewBuildPlannerModel(runner.auth, runner.svcModel)
	localCommit, err := bpm.FetchCommit(*branch.CommitID, params.project.Owner, params.project.Project, nil)
	if err != nil {
		return errs.Wrap(err, "Failed to fetch build result")
	}

	// Collect "after" buildplan
	bumpedBS, err := localCommit.BuildScript().Clone()
	if err != nil {
		return errs.Wrap(err, "Failed to clone build script")
	}

	now := captain.TimeValue{}
	now.Set("now")
	ts, err := commits_runbit.ExpandTime(&now, runner.auth)
	if err != nil {
		return errs.Wrap(err, "Failed to fetch latest timestamp")
	}
	bumpedBS.SetAtTime(ts, true)

	// Since our platform is commit based we need to create a commit for the "after" buildplan, even though we may not
	// end up using it it the user doesn't confirm the upgrade.
	bumpedCommit, err := bpm.StageCommitAndPoll(bpModel.StageCommitParams{
		Owner:        params.project.Owner,
		Project:      params.project.Project,
		ParentCommit: branch.CommitID.String(),
		Script:       bumpedBS,
	})
	if err != nil {
		return errs.Wrap(err, "Failed to stage bumped commit")
	}

	jsonBytes, err := json.Marshal(bumpedCommit)
	if err != nil {
		return fmt.Errorf("error marshaling results: %w", err)
	}
	runner.output.Print(string(jsonBytes))

	return nil
}
