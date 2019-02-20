package runtime

import (
	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/pkg/platform/model"
	"github.com/ActiveState/cli/pkg/platform/model/projects"
	"github.com/ActiveState/cli/pkg/project"
)

var (
	FailNoCommit = failures.Type("runtime.fail.nocommit")
)

type RuntimeDownload struct {
	project *project.Project
}

func InitRuntimeDownload(project *project.Project) *RuntimeDownload {
	return &RuntimeDownload{project}
}

func (r *RuntimeDownload) Download() *failures.Failure {
	platProject, fail := projects.FetchByName(r.project.Owner(), r.project.Name())
	if fail != nil {
		return fail
	}

	branch, fail := projects.DefaultBranch(platProject)
	if fail != nil {
		return fail
	}

	checkpoint, fail := model.FetchCheckpointForBranch(branch)
	if fail != nil {
		return fail
	}

	return nil
}
