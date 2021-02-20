package model

import (
	"context"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/pkg/platform/api/headchef"
	"github.com/ActiveState/cli/pkg/platform/api/headchef/headchef_models"
	"github.com/ActiveState/cli/pkg/platform/api/inventory/inventory_models"
	"github.com/ActiveState/cli/pkg/platform/model"
	"github.com/ActiveState/cli/pkg/platform/runtime2/build"
	"github.com/go-openapi/strfmt"
)

// var _ runtime.ClientProvider = &Default{}

// Default is the default client that actually talks to the backend
type Default struct{}

// NewDefault is the constructor for the Default client
func NewDefault() *Default {
	return &Default{}
}

func (d *Default) FetchCheckpointForCommit(commitID strfmt.UUID) (model.Checkpoint, strfmt.DateTime, error) {
	return model.FetchCheckpointForCommit(commitID)
}

func (d *Default) ResolveRecipe(commitID strfmt.UUID, owner, projectName string) (*inventory_models.Recipe, error) {
	return model.ResolveRecipe(commitID, owner, projectName)
}

func (d *Default) RequestBuild(recipeID, commitID strfmt.UUID, owner, project string) (headchef.BuildStatusEnum, *headchef_models.BuildStatusResponse, error) {
	return model.RequestBuild(recipeID, commitID, owner, project)
}

func (d *Default) BuildLog(ctx context.Context, artifactMap map[build.ArtifactID]build.Artifact, msgHandler build.BuildLogMessageHandler, recipeID strfmt.UUID) (*build.BuildLog, error) {
	conn, err := model.ConnectToBuildLogStreamer(ctx)
	if err != nil {
		return nil, errs.Wrap(err, "Could not get build updates")
	}

	return build.NewBuildLog(artifactMap, conn, msgHandler, recipeID)
}
