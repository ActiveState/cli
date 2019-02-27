package model

import (
	"time"

	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/pkg/platform/api/client/version_control"
	"github.com/ActiveState/cli/pkg/platform/api/inventory/inventory_models"
	"github.com/ActiveState/cli/pkg/platform/api/models"
	"github.com/ActiveState/cli/pkg/platform/authentication"
	"github.com/go-openapi/strfmt"
)

var (
	FailNoCommit      = failures.Type("model.fail.nocommit")
	FailGetCheckpoint = failures.Type("model.fail.getcheckpoint")
)

type Checkpoint []*models.Checkpoint

func FetchCheckpointForBranch(branch *models.Branch) (Checkpoint, *failures.Failure) {
	if branch.CommitID == nil {
		return nil, FailNoCommit.New(locale.T("err_no_commit"))
	}

	return FetchCheckpointForCommit(*branch.CommitID)
}

func FetchCheckpointForCommit(commitID strfmt.UUID) (Checkpoint, *failures.Failure) {
	auth := authentication.Get()
	params := version_control.NewGetCheckpointParams()
	params.CommitID = commitID

	response, err := auth.Client().VersionControl.GetCheckpoint(params, auth.ClientAuth())
	if err != nil {
		return nil, FailGetCheckpoint.New(locale.Tr("err_get_checkpoint", err.Error()))
	}

	return response.Payload, nil
}

func CheckpointToOrder(checkpoint Checkpoint) *inventory_models.Order {
	timestamp := strfmt.DateTime(time.Now())
	return &inventory_models.Order{
		Platforms:    CheckpointToPlatforms(checkpoint),
		Requirements: CheckpointToRequirements(checkpoint),
		Timestamp:    &timestamp,
	}
}

func CheckpointToRequirements(checkpoint Checkpoint) []*inventory_models.OrderRequirementsItems0 {
	result := []*inventory_models.OrderRequirementsItems0{}

	for _, req := range checkpoint {
		if req.Namespace == NamespacePlatform {
			continue
		}
		result = append(result, &inventory_models.OrderRequirementsItems0{
			PackageName:      &req.Requirement,
			Namespace:        req.Namespace,
			VersionSpecifier: req.VersionConstraint,
		})
	}

	return result
}

func CheckpointToPlatforms(checkpoint Checkpoint) []strfmt.UUID {
	result := []strfmt.UUID{}

	for _, req := range checkpoint {
		if req.Namespace != NamespacePlatform {
			continue
		}
		result = append(result, strfmt.UUID(req.Requirement))
	}

	return result
}
