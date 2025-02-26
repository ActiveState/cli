package model

import (
	"strings"

	"github.com/ActiveState/cli/pkg/buildscript"
	"github.com/ActiveState/cli/pkg/platform/api/mono/mono_models"
	"github.com/go-openapi/strfmt"

	"github.com/ActiveState/cli/pkg/sysinfo"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/pkg/platform/api/graphql"
	gqlModel "github.com/ActiveState/cli/pkg/platform/api/graphql/model"
	"github.com/ActiveState/cli/pkg/platform/api/graphql/request"
	"github.com/ActiveState/cli/pkg/platform/api/inventory/inventory_models"
	"github.com/ActiveState/cli/pkg/platform/authentication"
)

var (
	ErrNoData = errs.New("no data")
)

// Checkpoint represents a collection of requirements
type Checkpoint []*mono_models.Checkpoint

// Language represents a language requirement
type Language struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

// GetRequirement searches a commit for a requirement by name.
func GetRequirement(commitID strfmt.UUID, namespace Namespace, requirement string, auth *authentication.Auth) (*gqlModel.Requirement, error) {
	chkPt, err := FetchCheckpointForCommit(commitID, auth)
	if err != nil {
		return nil, err
	}

	chkPt = FilterCheckpointNamespace(chkPt, namespace.Type())

	for _, req := range chkPt {
		if req.Namespace == namespace.String() && req.Requirement == requirement {
			return req, nil
		}
	}

	return nil, nil
}

// FetchLanguagesForCommit fetches a list of language names for the given commit
func FetchLanguagesForCommit(commitID strfmt.UUID, auth *authentication.Auth) ([]Language, error) {
	checkpoint, err := FetchCheckpointForCommit(commitID, auth)
	if err != nil {
		return nil, err
	}

	languages := []Language{}
	for _, requirement := range checkpoint {
		if NamespaceMatch(requirement.Namespace, NamespaceLanguageMatch) {
			version := MonoConstraintsToString(requirement.VersionConstraints, true)
			lang := Language{
				Name:    requirement.Requirement,
				Version: version,
			}
			languages = append(languages, lang)
		}
	}

	return languages, nil
}

// FetchLanguagesForBuildScript fetches a list of language names for the given buildscript
func FetchLanguagesForBuildScript(script *buildscript.BuildScript) ([]Language, error) {
	languages := []Language{}
	reqs, err := script.DependencyRequirements()
	if err != nil {
		return nil, errs.Wrap(err, "failed to get dependency requirements")
	}

	for _, requirement := range reqs {
		if NamespaceMatch(requirement.Namespace, NamespaceLanguageMatch) {
			lang := Language{
				Name:    requirement.Name,
				Version: VersionRequirementsToString(requirement.VersionRequirement, true),
			}
			languages = append(languages, lang)
		}
	}

	return languages, nil
}

// FetchCheckpointForCommit fetches the checkpoint for the given commit
func FetchCheckpointForCommit(commitID strfmt.UUID, auth *authentication.Auth) ([]*gqlModel.Requirement, error) {
	logging.Debug("fetching checkpoint (%s)", commitID.String())

	request := request.CheckpointByCommit(commitID)

	gql := graphql.New(auth)
	response := []*gqlModel.Requirement{}
	err := gql.Run(request, &response)
	if err != nil {
		return nil, errs.Wrap(err, "gql.Run failed")
	}

	logging.Debug("Returning %d requirements", len(response))

	if len(response) == 0 {
		return nil, locale.WrapError(ErrNoData, "err_no_data_found")
	}

	return response, nil
}

func FetchAtTimeForCommit(commitID strfmt.UUID, auth *authentication.Auth) (strfmt.DateTime, error) {
	logging.Debug("fetching atTime for commit (%s)", commitID.String())

	request := request.CommitByID(commitID)

	gql := graphql.New(auth)
	response := gqlModel.Commit{}
	err := gql.Run(request, &response)
	if err != nil {
		return strfmt.DateTime{}, errs.Wrap(err, "gql.Run failed")
	}

	logging.Debug("Returning %s", response.AtTime)

	return response.AtTime, nil
}

func GqlReqsToMonoCheckpoint(requirements []*gqlModel.Requirement) []*mono_models.Checkpoint {
	var result = make([]*mono_models.Checkpoint, 0)
	for _, req := range requirements {
		result = append(result, &req.Checkpoint)
	}
	return result
}

// FilterCheckpointNamespace filters a Checkpoint removing requirements that do not match the given namespace.
func FilterCheckpointNamespace(chkPt []*gqlModel.Requirement, nsType ...NamespaceType) []*gqlModel.Requirement {
	if chkPt == nil {
		return nil
	}

	checkpoint := []*gqlModel.Requirement{}
	for _, ns := range nsType {
		for _, requirement := range chkPt {
			if NamespaceMatch(requirement.Namespace, ns.Matchable()) {
				checkpoint = append(checkpoint, requirement)
			}
		}
	}

	return checkpoint
}

// CheckpointToRequirements converts a checkpoint to a list of requirements for use with the head-chef
func CheckpointToRequirements(checkpoint Checkpoint) []*inventory_models.OrderRequirement {
	result := []*inventory_models.OrderRequirement{}

	for _, req := range checkpoint {
		if NamespaceMatch(req.Namespace, NamespacePlatformMatch) {
			continue
		}
		if NamespaceMatch(req.Namespace, NamespaceCamelFlagsMatch) {
			continue
		}

		result = append(result, &inventory_models.OrderRequirement{
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
func versionRequirement(versionConstraint string) []*inventory_models.VersionRequirement {
	if versionConstraint == "" {
		return nil
	}

	var eq = "eq"
	return []*inventory_models.VersionRequirement{{
		Comparator: &eq,
		Version:    &versionConstraint,
	}}
}

// CheckpointToPlatforms strips platforms from a checkpoint
func CheckpointToPlatforms(requirements []*gqlModel.Requirement) []strfmt.UUID {
	result := []strfmt.UUID{}

	for _, req := range requirements {
		if !NamespaceMatch(req.Namespace, NamespacePlatformMatch) {
			continue
		}
		result = append(result, strfmt.UUID(req.Requirement))
	}

	return result
}

func PlatformNameToPlatformID(name string) (string, error) {
	name = strings.ToLower(name)
	if name == "darwin" {
		name = "macos"
	}
	switch strings.ToLower(name) {
	case strings.ToLower(sysinfo.Linux.String()):
		return constants.LinuxBit64UUID, nil
	case strings.ToLower(sysinfo.Mac.String()):
		return constants.MacBit64UUID, nil
	case strings.ToLower(sysinfo.Windows.String()):
		return constants.Win10Bit64UUID, nil
	default:
		return "", ErrPlatformNotFound
	}
}

func HostPlatformToKernelName(os string) string {
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
		case "arm":
			return "arm64"
		}
	}
	return "unrecognized"
}

func fallbackArch(platform, arch string) string {
	// On the M1 Mac platform we default to
	// amd64 as the platform does not support arm.
	if arch == "arm64" && platform == sysinfo.Mac.String() {
		return "amd64"
	}
	return arch
}
