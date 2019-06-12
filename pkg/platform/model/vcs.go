package model

import (
	"regexp"
	"time"

	"github.com/go-openapi/strfmt"

	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/pkg/platform/api/inventory/inventory_models"
	"github.com/ActiveState/cli/pkg/platform/api/mono/mono_client/version_control"
	mono_models "github.com/ActiveState/cli/pkg/platform/api/mono/mono_models"
	"github.com/ActiveState/cli/pkg/platform/authentication"
)

var (
	// FailNoCommit is a failure due to a non-existent commit
	FailNoCommit = failures.Type("model.fail.nocommit")

	// FailGetCheckpoint is a failure in the call to api.GetCheckpoint
	FailGetCheckpoint = failures.Type("model.fail.getcheckpoint")
)

// Checkpoint represents a collection of requirements
type Checkpoint []*mono_models.Checkpoint

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
		if NamespaceMatch(requirement.Namespace, NamespaceLanguage) {
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
		if NamespaceMatch(req.Namespace, NamespacePlatform) {
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
		if !NamespaceMatch(req.Namespace, NamespacePlatform) {
			continue
		}
		result = append(result, strfmt.UUID(req.Requirement))
	}

	return result
}

// Namespace represents regular expression strings used for defining matchable
// requirements.
type Namespace string

const (
	// NamespacePlatform is the namespace used for platform requirements
	NamespacePlatform Namespace = `^platform$`

	// NamespaceLanguage is the namespace used for language requirements
	NamespaceLanguage = `^language$`

	// NamespacePackage is the namespace used for package requirements
	NamespacePackage = `/package$`
)

// NamespaceMatch Checks if the given namespace query matches the given namespace
func NamespaceMatch(query string, namespace Namespace) bool {
	match, err := regexp.Match(string(namespace), []byte(query))
	if err != nil {
		logging.Error("Could not match regex for %v, query: %s, error: %v", namespace, query, err)
	}
	return match
}

// LatestCommitID returns the latest commit id by owner and project names. It
// possible for a nil commit id to be returned without failure.
func LatestCommitID(ownerName, projectName string) (*strfmt.UUID, *failures.Failure) {
	proj, fail := FetchProjectByName(ownerName, projectName)
	if fail != nil {
		return nil, fail
	}

	branch, fail := DefaultBranchForProject(proj)
	if fail != nil {
		return nil, fail
	}

	return branch.CommitID, nil
}
