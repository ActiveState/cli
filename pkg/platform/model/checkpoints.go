package model

import (
	"strings"

	"github.com/go-openapi/strfmt"

	"github.com/ActiveState/sysinfo"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/pkg/platform/api"
	"github.com/ActiveState/cli/pkg/platform/api/graphql"
	"github.com/ActiveState/cli/pkg/platform/api/graphql/model"
	"github.com/ActiveState/cli/pkg/platform/api/graphql/request"
	"github.com/ActiveState/cli/pkg/platform/api/inventory/inventory_models"
	"github.com/ActiveState/cli/pkg/platform/api/mono/mono_models"
)

var (
	// FailGetCheckpoint is a failure in the call to api.GetCheckpoint
	FailGetCheckpoint = failures.Type("model.fail.getcheckpoint")

	// FailNoData represents an error due to lacking returned data
	FailNoData = failures.Type("model.fail.nodata", failures.FailNonFatal)
)

// Checkpoint represents a collection of requirements
type Checkpoint []*model.Requirement

// Language represents a langauge requirement
type Language struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

// HasPackage searches a commit for a package by name.
func HasPackage(commitID strfmt.UUID, namespace string) (bool, error) {
	chkPt, _, fail := FetchCheckpointForCommit(commitID)
	if fail != nil {
		return false, fail.ToError()
	}

	chkPt = FilterCheckpointPackages(chkPt)

	for _, req := range chkPt {
		if req.Requirement == namespace {
			return true, nil
		}
	}

	return false, nil
}

// FetchLanguagesForProject fetches a list of language names for the given project
func FetchLanguagesForProject(orgName string, projectName string) ([]Language, *failures.Failure) {
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
func FetchLanguagesForBranch(branch *mono_models.Branch) ([]Language, *failures.Failure) {
	if branch.CommitID == nil {
		return nil, FailNoCommit.New(locale.T("err_no_commit"))
	}

	return FetchLanguagesForCommit(*branch.CommitID)
}

// FetchLanguagesForCommit fetches a list of language names for the given commit
func FetchLanguagesForCommit(commitID strfmt.UUID) ([]Language, *failures.Failure) {
	checkpoint, _, fail := FetchCheckpointForCommit(commitID)
	if fail != nil {
		return nil, fail
	}

	languages := []Language{}
	for _, requirement := range checkpoint {
		if NamespaceMatch(requirement.Namespace, NamespaceLanguageMatch) {
			languages = append(languages, Language{
				Name:    requirement.Requirement,
				Version: requirement.VersionConstraint,
			})
		}
	}

	return languages, nil
}

// FetchCheckpointForCommit fetches the checkpoint for the given commit
func FetchCheckpointForCommit(commitID strfmt.UUID) (Checkpoint, strfmt.DateTime, *failures.Failure) {
	logging.Debug("fetching checkpoint (%s)", commitID.String())

	request := request.CheckpointByCommit(commitID)

	gql := graphql.Get()
	response := model.Checkpoint{}
	err := gql.Run(request, &response)
	if err != nil {
		return nil, strfmt.DateTime{}, api.FailUnknown.Wrap(err)
	}

	logging.Debug("Returning %d requirements", len(response.Requirements))

	if response.Commit == nil {
		return nil, strfmt.DateTime{}, FailNoData.New(locale.T("err_no_data_found"))
	}

	return response.Requirements, response.Commit.AtTime, nil
}

// FilterCheckpointPackages filters a Checkpoint removing requirements that
// are not packages. If nil data is provided, a nil slice is returned. If no
// packages remain after filtering, an empty slice is returned.
func FilterCheckpointPackages(chkPt Checkpoint) Checkpoint {
	if chkPt == nil {
		return nil
	}

	checkpoint := Checkpoint{}
	for _, requirement := range chkPt {
		if !NamespaceMatch(requirement.Namespace, NamespacePackageMatch) && !NamespaceMatch(requirement.Namespace, NamespaceBundlesMatch) {
			continue
		}

		checkpoint = append(checkpoint, requirement)
	}

	return checkpoint
}

// CheckpointToRequirements converts a checkpoint to a list of requirements for use with the head-chef
func CheckpointToRequirements(checkpoint Checkpoint) []*inventory_models.V1OrderRequirementsItems {
	result := []*inventory_models.V1OrderRequirementsItems{}

	for _, req := range checkpoint {
		if NamespaceMatch(req.Namespace, NamespacePlatformMatch) {
			continue
		}
		if NamespaceMatch(req.Namespace, NamespaceCamelFlagsMatch) {
			continue
		}

		result = append(result, &inventory_models.V1OrderRequirementsItems{
			Feature:             &req.Requirement,
			Namespace:           &req.Namespace,
			VersionRequirements: versionRequirement(req.VersionConstraint),
		})
	}

	return result
}

// CheckpointToCamelFlags converts a checkpoint to camel flags
func CheckpointToCamelFlags(checkpoint Checkpoint) []string {
	result := []string{}

	for _, req := range checkpoint {
		if !NamespaceMatch(req.Namespace, NamespaceCamelFlagsMatch) {
			continue
		}

		result = append(result, req.Requirement)
	}

	return result
}

// versionRequirement returns nil if the version constraint is empty otherwise it will return a valid
// list for a V1OrderRequirements' VersionRequirements. The VersionRequirements can be omitted however
// if it is present then the Version string must be populated with at least one character.
func versionRequirement(versionConstraint string) []*inventory_models.V1OrderRequirementsItemsVersionRequirementsItems {
	if versionConstraint == "" {
		return nil
	}

	var eq = "eq"
	return []*inventory_models.V1OrderRequirementsItemsVersionRequirementsItems{{
		Comparator: &eq,
		Version:    &versionConstraint,
	}}
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

// CheckpointToLanguage returns the language from a checkpoint
func CheckpointToLanguage(checkpoint Checkpoint) (*Language, *failures.Failure) {
	for _, req := range checkpoint {
		if !NamespaceMatch(req.Namespace, NamespaceLanguageMatch) {
			continue
		}
		lang, fail := FetchLanguageByDetails(req.Requirement, req.VersionConstraint)
		if fail != nil {
			return nil, fail
		}
		return lang, nil
	}

	return nil, failures.FailNotFound.New(locale.T("err_fetch_languages"))
}

func hostPlatformToPlatformID(os string) (string, *failures.Failure) {
	switch strings.ToLower(os) {
	case strings.ToLower(sysinfo.Linux.String()):
		return constants.LinuxBit64UUID, nil
	case strings.ToLower(sysinfo.Mac.String()):
		return constants.MacBit64UUID, nil
	case strings.ToLower(sysinfo.Windows.String()):
		return constants.Win10Bit64UUID, nil
	default:
		return "", FailUnsupportedPlatform.New("err_unsupported_platform", os)
	}
}

func hostPlatformToKernelName(os string) string {
	switch strings.ToLower(os) {
	case strings.ToLower(sysinfo.Linux.String()):
		return "Linux"
	case strings.ToLower(sysinfo.Mac.String()):
		return "Darwin"
	case strings.ToLower(sysinfo.Windows.String()):
		return "Windows"
	default:
		return ""
	}
}

func platformArchToHostArch(arch, bits string) string {
	switch bits {
	case "32":
		switch arch {
		case "IA64":
			return "nonexistent"
		case "PA-RISC":
			return "unsupported"
		case "PowerPC":
			return "ppc"
		case "Sparc":
			return "sparc"
		case "x86":
			return "386"
		}
	case "64":
		switch arch {
		case "IA64":
			return "unsupported"
		case "PA-RISC":
			return "unsupported"
		case "PowerPC":
			return "ppc64"
		case "Sparc":
			return "sparc64"
		case "x86":
			return "amd64"
		}
	}
	return "unrecognized"
}
