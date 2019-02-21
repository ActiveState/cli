package runtime

import (
	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/pkg/platform/api/headchef"
	"github.com/ActiveState/cli/pkg/platform/api/headchef/headchef_models"
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

	recipes, fail := model.FetchRecipesForProject(platProject)
	if fail != nil {
		return fail
	}

	effectiveRecipe, fail := model.EffectiveRecipe(recipes)
	if fail != nil {
		return fail
	}

	buildRecipe, fail := model.RecipeToBuildRecipe(effectiveRecipe)
	if fail != nil {
		return fail
	}

	buildRequestor, fail := model.BuildRequestorForProject(platProject)
	if fail != nil {
		return fail
	}

	done := make(chan bool)

	request := headchef.NewRequest(buildRecipe, buildRequestor)
	request.OnBuildCompleted(func(response headchef_models.BuildCompleted) {
		logging.Debug("Build completed: %v", response.Artifacts[0])
	})

	request.OnBuildStarted(func() {
		logging.Debug("Build started")
	})

	request.OnBuildFailed(func(message string) {
		logging.Debug("Build failed: %s", message)
	})

	request.OnFailure(func(fail *failures.Failure) {
		logging.Debug("Failure: %v", fail)
	})

	request.OnClose(func() {
		logging.Debug("Done")
		done <- true
	})

	request.Start()

	<-done

	return nil
}
