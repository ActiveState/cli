package model

import (
	"time"

	"github.com/ActiveState/cli/pkg/platform/api"
	"github.com/ActiveState/cli/pkg/platform/api/graphql"
	"github.com/ActiveState/cli/pkg/platform/api/graphql/model"
	"github.com/ActiveState/cli/pkg/platform/api/graphql/request"

	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/pkg/platform/api/inventory/inventory_models"
	mono_models "github.com/ActiveState/cli/pkg/platform/api/mono/mono_models"
	"github.com/go-openapi/strfmt"
)

var (
	// FailGetCheckpoint is a failure in the call to api.GetCheckpoint
	FailGetCheckpoint = failures.Type("model.fail.getcheckpoint")
)

// Checkpoint represents a collection of requirements
type Checkpoint []*model.Requirement

// FetchLanguagesForBranch fetches a list of language names for the given branch
func FetchLanguagesForBranch(branch *mono_models.Branch) ([]string, *failures.Failure) {
	if branch.CommitID == nil {
		return nil, FailNoCommit.New(locale.T("err_no_commit"))
	}

	return FetchLanguagesForCommit(*branch.CommitID)
}

// FetchLanguagesForCommit fetches a list of language names for the given commit
func FetchLanguagesForCommit(commitID strfmt.UUID) ([]string, *failures.Failure) {
	checkpoint, fail := FetchCheckpointForCommit(commitID)
	if fail != nil {
		return nil, fail
	}

	languages := []string{}
	for _, requirement := range checkpoint {
		if NamespaceMatch(requirement.Namespace, NamespaceLanguageMatch) {
			languages = append(languages, requirement.Requirement)
		}
	}

	return languages, nil
}

// FetchCheckpointForCommit fetches the checkpoint for the given commit
func FetchCheckpointForCommit(commitID strfmt.UUID) (Checkpoint, *failures.Failure) {
	logging.Debug("fetching checkpoint (%s)", commitID.String())

	request := request.CheckpointByCommit(commitID)

	gql := graphql.Get()
	response := model.Checkpoint{}
	err := gql.Run(request, &response)
	if err != nil {
		return nil, api.FailUnknown.Wrap(err)
	}

	logging.Debug("Returning %d requirements", len(response.Requirements))

	return response.Requirements, nil
}

// CheckpointToOrder converts a checkpoint to an order
func CheckpointToOrder(checkpoint Checkpoint) *inventory_models.Order {
	timestamp := strfmt.DateTime(time.Now())
	return &inventory_models.Order{
		Platforms:    CheckpointToPlatforms(checkpoint),
		Requirements: CheckpointToRequirements(checkpoint),
		Timestamp:    &timestamp,
	}
}

// CheckpointToRequirements converts a checkpoint to a list of requirements for use with the head-chef
func CheckpointToRequirements(checkpoint Checkpoint) []*inventory_models.OrderRequirementsItems0 {
	result := []*inventory_models.OrderRequirementsItems0{}

	for _, req := range checkpoint {
		if NamespaceMatch(req.Namespace, NamespacePlatformMatch) {
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

// CheckpointToPlatforms strips platforms from a checkpoint
func CheckpointToPlatforms(checkpoint Checkpoint) []strfmt.UUID {
	result := []strfmt.UUID{}

	for _, req := range checkpoint {
		if !NamespaceMatch(req.Namespace, NamespacePlatformMatch) {
			continue
		}
		result = append(result, strfmt.UUID(req.Requirement))
	}

	return result
}
