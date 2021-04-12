package model

import (
	"strings"

	"github.com/go-openapi/strfmt"

	"github.com/ActiveState/sysinfo"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/pkg/platform/api/graphql"
	"github.com/ActiveState/cli/pkg/platform/api/graphql/model"
	"github.com/ActiveState/cli/pkg/platform/api/graphql/request"
	"github.com/ActiveState/cli/pkg/platform/api/inventory/inventory_models"
)

var (
	ErrNoData = errs.New("no data")
)

// Checkpoint represents a collection of requirements
type Checkpoint []*model.Requirement

// Language represents a langauge requirement
type Language struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

// GetRequirement searches a commit for a requirement by name.
func GetRequirement(commitID strfmt.UUID, namespace, requirement string) (*model.Requirement, error) {
	chkPt, _, err := FetchCheckpointForCommit(commitID)
	if err != nil {
		return nil, err
	}

	chkPt = FilterCheckpointPackages(chkPt)

	for _, req := range chkPt {
		if req.Namespace == namespace && req.Requirement == requirement {
			return req, nil
		}
	}

	return nil, nil
}

// FetchLanguagesForCommit fetches a list of language names for the given commit
func FetchLanguagesForCommit(commitID strfmt.UUID) ([]Language, error) {
	checkpoint, _, err := FetchCheckpointForCommit(commitID)
	if err != nil {
		return nil, err
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
func FetchCheckpointForCommit(commitID strfmt.UUID) (Checkpoint, strfmt.DateTime, error) {
	logging.Debug("fetching checkpoint (%s)", commitID.String())

	request := request.CheckpointByCommit(commitID)

	gql := graphql.New()
	response := model.Checkpoint{}
	err := gql.Run(request, &response)
	if err != nil {
		return nil, strfmt.DateTime{}, errs.Wrap(err, "gql.Run failed")
	}

	logging.Debug("Returning %d requirements", len(response.Requirements))

	if response.Commit == nil {
		return nil, strfmt.DateTime{}, locale.WrapError(ErrNoData, "err_no_data_found")
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
func CheckpointToLanguage(checkpoint Checkpoint) (*Language, error) {
	for _, req := range checkpoint {
		if !NamespaceMatch(req.Namespace, NamespaceLanguageMatch) {
			continue
		}
		lang, err := FetchLanguageByDetails(req.Requirement, req.VersionConstraint)
		if err != nil {
			return nil, err
		}
		return lang, nil
	}

	return nil, locale.NewError("err_fetch_languages")
}

func PlatformNameToPlatformID(name string) (string, error) {
	name = strings.ToLower(name)
	if name == "darwin" {
		name = "macos"
	}
	id, err := hostPlatformToPlatformID(name)
	return id, err
}

func hostPlatformToPlatformID(os string) (string, error) {
	switch strings.ToLower(os) {
	case strings.ToLower(sysinfo.Linux.String()):
		return constants.LinuxBit64UUID, nil
	case strings.ToLower(sysinfo.Mac.String()):
		return constants.MacBit64UUID, nil
	case strings.ToLower(sysinfo.Windows.String()):
		return constants.Win10Bit64UUID, nil
	default:
		return "", locale.NewInputError("err_unsupported_platform", "", os)
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
