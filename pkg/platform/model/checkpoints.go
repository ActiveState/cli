package model

import (
	"time"

	"github.com/go-openapi/strfmt"

	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/pkg/platform/api/inventory/inventory_models"
	"github.com/ActiveState/cli/pkg/platform/api/mono/mono_client/version_control"
	mono_models "github.com/ActiveState/cli/pkg/platform/api/mono/mono_models"
	"github.com/ActiveState/cli/pkg/platform/authentication"
)

var (
	// FailGetCheckpoint is a failure in the call to api.GetCheckpoint
	FailGetCheckpoint = failures.Type("model.fail.getcheckpoint")
)

// Checkpoint represents a collection of requirements
type Checkpoint []*mono_models.Checkpoint

// FetchLanguagesForProject fetches a list of language names for the given project
func FetchLanguagesForProject(orgName string, projectName string) ([]string, *failures.Failure) {
	platProject, fail := FetchProjectByName(orgName, projectName)
	if fail != nil {
		return nil, fail
	}

	branch, fail := DefaultBranchForProject(platProject)
	if fail != nil {
		return nil, fail
	}

	return FetchLanguagesForBranch(branch)
}

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

// FetchCheckpointForBranch fetches the checkpoint for the given branch
func FetchCheckpointForBranch(branch *mono_models.Branch) (Checkpoint, *failures.Failure) {
	if branch.CommitID == nil {
		return nil, FailNoCommit.New(locale.T("err_no_commit"))
	}

	return FetchCheckpointForCommit(*branch.CommitID)
}

// FetchCheckpointForCommit fetches the checkpoint for the given commit
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

// CheckpointToOrder converts a checkpoint to an order
func CheckpointToOrder(commitID strfmt.UUID, checkpoint Checkpoint) *inventory_models.V1Order {
	timestamp := strfmt.DateTime(time.Now())
	return &inventory_models.V1Order{
		OrderID:      &commitID,
		Platforms:    CheckpointToPlatforms(checkpoint),
		Requirements: CheckpointToRequirements(checkpoint),
		Timestamp:    &timestamp,
	}
}

// CheckpointToRequirements converts a checkpoint to a list of requirements for use with the head-chef
func CheckpointToRequirements(checkpoint Checkpoint) []*inventory_models.V1OrderRequirementsItems {
	result := []*inventory_models.V1OrderRequirementsItems{}

	for _, req := range checkpoint {
		if NamespaceMatch(req.Namespace, NamespacePlatformMatch) {
			continue
		}

		eq := "eq"
		result = append(result, &inventory_models.V1OrderRequirementsItems{
			Feature:   &req.Requirement,
			Namespace: &req.Namespace,
			VersionRequirements: []*inventory_models.V1OrderRequirementsItemsVersionRequirementsItems{{
				Comparator: &eq,
				Version:    &req.VersionConstraint,
			}},
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
