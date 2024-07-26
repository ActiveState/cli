package requirements

import (
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/ActiveState/cli/internal/analytics"
	"github.com/ActiveState/cli/internal/captain"
	"github.com/ActiveState/cli/internal/config"
	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/primer"
	"github.com/ActiveState/cli/internal/prompt"
	"github.com/ActiveState/cli/internal/runbits/buildscript"
	"github.com/ActiveState/cli/pkg/localcommit"
	"github.com/ActiveState/cli/pkg/platform/api/buildplanner/response"
	"github.com/ActiveState/cli/pkg/platform/api/buildplanner/types"
	"github.com/ActiveState/cli/pkg/platform/authentication"
	"github.com/ActiveState/cli/pkg/platform/model"
	bpModel "github.com/ActiveState/cli/pkg/platform/model/buildplanner"
	"github.com/ActiveState/cli/pkg/project"
	"github.com/ActiveState/cli/pkg/runtime"
	"github.com/go-openapi/strfmt"
)

type PackageVersion struct {
	captain.NameVersionValue
}

func (pv *PackageVersion) Set(arg string) error {
	err := pv.NameVersionValue.Set(arg)
	if err != nil {
		return locale.WrapInputError(err, "err_package_format", "The package and version provided is not formatting correctly. It must be in the form of <package>@<version>")
	}
	return nil
}

type RequirementOperation struct {
	prime primeable
	// The remainder is redundant with the above. Refactoring this will follow in a later story so as not to blow
	// up the one that necessitates adding the primer at this level.
	// https://activestatef.atlassian.net/browse/DX-2869
	Output    output.Outputer
	Prompt    prompt.Prompter
	Project   *project.Project
	Auth      *authentication.Auth
	Config    *config.Instance
	Analytics analytics.Dispatcher
	SvcModel  *model.SvcModel
}

type primeable interface {
	primer.Outputer
	primer.Prompter
	primer.Projecter
	primer.Auther
	primer.Configurer
	primer.Analyticer
	primer.SvcModeler
}

func NewRequirementOperation(prime primeable) *RequirementOperation {
	return &RequirementOperation{
		prime,
		prime.Output(),
		prime.Prompt(),
		prime.Project(),
		prime.Auth(),
		prime.Config(),
		prime.Analytics(),
		prime.SvcModel(),
	}
}

const latestVersion = "latest"

type ErrNoMatches struct {
	*locale.LocalizedError
	Query        string
	Alternatives *string
}

var errNoRequirements = errs.New("No requirements were provided")

var errInitialNoRequirement = errs.New("Could not find compatible requirement for initial commit")

var errNoLanguage = errs.New("No language")

var versionRe = regexp.MustCompile(`^\d(\.\d+)*$`)

// Requirement represents a package, language or platform requirement
type Requirement struct {
	Name      string
	Version   string
	Revision  *int
	BitWidth  int // Only needed for platform requirements
	Namespace *model.Namespace
	Operation types.Operation
}

type ResolveNamespaceError struct {
	error
	Name string
}

func (r *RequirementOperation) updateCommitID(commitID strfmt.UUID) error {
	if err := localcommit.Set(r.Project.Dir(), commitID.String()); err != nil {
		return locale.WrapError(err, "err_package_update_commit_id")
	}

	if r.Config.GetBool(constants.OptinBuildscriptsConfig) {
		bp := bpModel.NewBuildPlannerModel(r.Auth)
		script, err := bp.GetBuildScript(commitID.String())
		if err != nil {
			return errs.Wrap(err, "Could not get remote build expr and time")
		}

		err = buildscript_runbit.Update(r.Project, script)
		if err != nil {
			return locale.WrapError(err, "err_update_build_script")
		}
	}

	return nil
}

func getSuggestions(namespace *model.Namespace, name string, auth *authentication.Auth) ([]string, error) {
	ns := ""
	if namespace != nil {
		ns = namespace.String()
	}
	results, err := model.SearchIngredients(ns, name, false, nil, auth)
	if err != nil {
		return []string{}, locale.WrapError(err, "package_ingredient_err_search", "Failed to resolve ingredient named: {{.V0}}", name)
	}

	maxResults := 5
	if len(results) > maxResults {
		results = results[:maxResults]
	}

	suggestions := make([]string, 0, maxResults+1)
	for _, result := range results {
		suggestions = append(suggestions, fmt.Sprintf(" - %s", *result.Ingredient.Name))
	}

	return suggestions, nil
}

func commitMessage(requirements ...*Requirement) string {
	switch len(requirements) {
	case 0:
		return ""
	case 1:
		return requirementCommitMessage(requirements[0])
	default:
		return commitMessageMultiple(requirements...)
	}
}

func requirementCommitMessage(req *Requirement) string {
	switch req.Namespace.Type() {
	case model.NamespaceLanguage:
		return languageCommitMessage(req.Operation, req.Name, req.Version)
	case model.NamespacePlatform:
		return platformCommitMessage(req.Operation, req.Name, req.Version, req.BitWidth)
	case model.NamespacePackage, model.NamespaceBundle:
		return packageCommitMessage(req.Operation, req.Name, req.Version)
	}
	return ""
}

func languageCommitMessage(op types.Operation, name, version string) string {
	var msgL10nKey string
	switch op {
	case types.OperationAdded:
		msgL10nKey = "commit_message_added_language"
	case types.OperationUpdated:
		msgL10nKey = "commit_message_updated_language"
	case types.OperationRemoved:
		msgL10nKey = "commit_message_removed_language"
	}

	return locale.Tr(msgL10nKey, name, version)
}

func platformCommitMessage(op types.Operation, name, version string, word int) string {
	var msgL10nKey string
	switch op {
	case types.OperationAdded:
		msgL10nKey = "commit_message_added_platform"
	case types.OperationUpdated:
		msgL10nKey = "commit_message_updated_platform"
	case types.OperationRemoved:
		msgL10nKey = "commit_message_removed_platform"
	}

	return locale.Tr(msgL10nKey, name, strconv.Itoa(word), version)
}

func packageCommitMessage(op types.Operation, name, version string) string {
	var msgL10nKey string
	switch op {
	case types.OperationAdded:
		msgL10nKey = "commit_message_added_package"
	case types.OperationUpdated:
		msgL10nKey = "commit_message_updated_package"
	case types.OperationRemoved:
		msgL10nKey = "commit_message_removed_package"
	}

	if version == "" {
		version = locale.Tl("package_version_auto", "auto")
	}
	return locale.Tr(msgL10nKey, name, version)
}

func commitMessageMultiple(requirements ...*Requirement) string {
	var commitDetails []string
	for _, req := range requirements {
		commitDetails = append(commitDetails, requirementCommitMessage(req))
	}

	return locale.Tl("commit_message_multiple", "Committing changes to multiple requirements: {{.V0}}", strings.Join(commitDetails, ", "))
}

func requirementNames(requirements ...*Requirement) []string {
	var names []string
	for _, requirement := range requirements {
		names = append(names, requirement.Name)
	}
	return names
}

func IsBuildError(err error) bool {
	var errBuild *runtime.BuildError
	var errBuildPlanner *response.BuildPlannerError

	return errors.As(err, &errBuild) || errors.As(err, &errBuildPlanner)
}
