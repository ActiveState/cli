package rebuildproject

import (
	"fmt"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/primer"
	"github.com/ActiveState/cli/pkg/platform/api/buildplanner/types"
	"github.com/ActiveState/cli/pkg/platform/authentication"
	"github.com/ActiveState/cli/pkg/platform/model"
	"github.com/ActiveState/cli/pkg/platform/model/buildplanner"
	bpModel "github.com/ActiveState/cli/pkg/platform/model/buildplanner"
	"github.com/ActiveState/cli/pkg/project"
)

type RebuildProjectRunner struct {
	auth     *authentication.Auth
	output   output.Outputer
	svcModel *model.SvcModel
}

func New(p *primer.Values) *RebuildProjectRunner {
	return &RebuildProjectRunner{
		auth:     p.Auth(),
		output:   p.Output(),
		svcModel: p.SvcModel(),
	}
}

type Params struct {
	Namespace *project.Namespaced
}

func NewParams() *Params {
	return &Params{}
}

func (runner *RebuildProjectRunner) Run(params *Params) error {
	branch, err := model.DefaultBranchForProjectName(params.Namespace.Owner, params.Namespace.Project)
	if err != nil {
		return fmt.Errorf("error fetching default branch: %w", err)
	}

	// Collect "before" buildscript
	bpm := bpModel.NewBuildPlannerModel(runner.auth, runner.svcModel)
	localCommit, err := bpm.FetchCommitNoPoll(*branch.CommitID, params.Namespace.Owner, params.Namespace.Project, nil)
	if err != nil {
		return errs.Wrap(err, "Failed to fetch build result")
	}

	// Collect "after" buildscript
	bumpedBS, err := localCommit.BuildScript().Clone()
	if err != nil {
		return errs.Wrap(err, "Failed to clone build script")
	}

	latest, err := model.FetchLatestRevisionTimeStamp(runner.auth)
	if err != nil {
		return errs.Wrap(err, "Failed to fetch latest timestamp")
	}
	bumpedBS.SetAtTime(latest, true)

	// Since our platform is commit based we need to create a commit for the "after" buildscript
	bumpedCommit, err := bpm.StageCommitAndPoll(bpModel.StageCommitParams{
		Owner:        params.Namespace.Owner,
		Project:      params.Namespace.Project,
		ParentCommit: branch.CommitID.String(),
		Script:       bumpedBS,
	})
	if err != nil {
		return errs.Wrap(err, "Failed to stage bumped commit")
	}

	// Now, merge the new commit using the branch name to fast-forward
	_, err = bpm.MergeCommit(&buildplanner.MergeCommitParams{
		Owner:     params.Namespace.Owner,
		Project:   params.Namespace.Project,
		TargetRef: branch.Label,
		OtherRef:  bumpedCommit.CommitID.String(),
		Strategy:  types.MergeCommitStrategyFastForward,
	})
	if err != nil {
		return fmt.Errorf("error merging commit: %w", err)
	}

	runner.output.Print("Project is now rebuilding with commit ID " + bumpedCommit.CommitID.String())

	return nil
}
