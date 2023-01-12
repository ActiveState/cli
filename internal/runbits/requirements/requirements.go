package requirements

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/ActiveState/cli/internal/analytics"
	anaConsts "github.com/ActiveState/cli/internal/analytics/constants"
	"github.com/ActiveState/cli/internal/captain"
	"github.com/ActiveState/cli/internal/config"
	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/installation/storage"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/multilog"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/prompt"
	"github.com/ActiveState/cli/internal/runbits"
	medmodel "github.com/ActiveState/cli/pkg/platform/api/mediator/model"
	"github.com/ActiveState/cli/pkg/platform/authentication"
	"github.com/ActiveState/cli/pkg/platform/model"
	"github.com/ActiveState/cli/pkg/platform/runtime/artifact"
	"github.com/ActiveState/cli/pkg/platform/runtime/target"
	"github.com/ActiveState/cli/pkg/project"
	"github.com/ActiveState/cli/pkg/projectfile"
	"github.com/go-openapi/strfmt"
	"github.com/thoas/go-funk"
)

type PackageVersion struct {
	captain.NameVersion
}

func (pv *PackageVersion) Set(arg string) error {
	err := pv.NameVersion.Set(arg)
	if err != nil {
		return locale.WrapInputError(err, "err_package_format", "The package and version provided is not formatting correctly, must be in the form of <package>@<version>")
	}
	return nil
}

const latestVersion = "latest"

type RequirementOperationParams struct {
	Output              output.Outputer
	Prompt              prompt.Prompter
	Project             *project.Project
	Auth                *authentication.Auth
	Config              *config.Instance
	Analytics           analytics.Dispatcher
	SvcModel            *model.SvcModel
	RequirementName     string
	RequirementVersion  string
	RequirementBitWidth int
	Operation           model.Operation
	NsType              model.NamespaceType
}

func ExecuteRequirementOperation(params *RequirementOperationParams) (rerr error) {
	var ns model.Namespace
	var langVersion string
	langName := "undetermined"

	out := params.Output
	var pg *output.DotProgress
	defer func() {
		if pg != nil && !pg.Stopped() {
			pg.Stop(locale.T("progress_fail"))
		}
	}()

	var err error
	pj := params.Project
	if pj == nil {
		pg = output.NewDotProgress(params.Output, locale.Tl("progress_project", "", params.RequirementName), 10*time.Second)
		pj, err = initializeProject()
		if err != nil {
			return locale.WrapError(err, "err_package_get_project", "Could not get project from path")
		}
		pg.Stop(locale.T("progress_success"))

		defer func() {
			if rerr != nil && !errors.Is(err, artifact.CamelRuntimeBuilding) {
				if err := os.Remove(pj.Source().Path()); err != nil {
					multilog.Error("could not remove temporary project file: %s", errs.JoinMessage(err))
				}
			}
		}()
	} else {
		out.Notice(locale.Tl("operating_message", "", pj.NamespaceString(), pj.Dir()))
	}

	switch params.NsType {
	case model.NamespacePackage, model.NamespaceBundle:
		if pj == nil {
			break
		}
		language, err := model.LanguageByCommit(pj.CommitUUID())
		if err == nil {
			langName = language.Name
			ns = model.NewNamespacePkgOrBundle(langName, params.NsType)
		} else {
			logging.Debug("Could not get language from project: %v", err)
		}
	case model.NamespaceLanguage:
		ns = model.NewNamespaceLanguage()
	case model.NamespacePlatform:
		ns = model.NewNamespacePlatform()
	}

	var validatePkg = params.Operation == model.OperationAdded && (ns.Type() == model.NamespacePackage || ns.Type() == model.NamespaceBundle)
	if !ns.IsValid() {
		pg = output.NewDotProgress(out, locale.Tl("progress_pkg_nolang", "", params.RequirementName), 10*time.Second)

		supported, err := model.FetchSupportedLanguages(model.HostPlatform)
		if err != nil {
			return errs.Wrap(err, "Failed to retrieve the list of supported languages")
		}

		var supportedLang *medmodel.SupportedLanguage
		params.RequirementName, ns, supportedLang, err = resolvePkgAndNamespace(params.Prompt, params.RequirementName, params.NsType, supported)
		if err != nil {
			return errs.Wrap(err, "Could not resolve pkg and namespace")
		}
		langVersion = supportedLang.DefaultVersion
		langName = supportedLang.Name

		validatePkg = false

		pg.Stop(locale.T("progress_found"))
	}

	if strings.ToLower(params.RequirementVersion) == latestVersion {
		params.RequirementVersion = ""
	}

	if validatePkg {
		pg = output.NewDotProgress(out, locale.Tl("progress_search", "", params.RequirementName), 10*time.Second)

		packages, err := model.SearchIngredientsStrict(ns, params.RequirementName, false, false)
		if err != nil {
			return locale.WrapError(err, "package_err_cannot_obtain_search_results")
		}
		if len(packages) == 0 {
			suggestions, err := getSuggestions(ns, params.RequirementName)
			if err != nil {
				multilog.Error("Failed to retrieve suggestions: %v", err)
			}
			if len(suggestions) == 0 {
				return locale.WrapInputError(err, "package_ingredient_alternatives_nosuggest", "", params.RequirementName)
			}
			return locale.WrapInputError(err, "package_ingredient_alternatives", "", params.RequirementName, strings.Join(suggestions, "\n"))
		}

		pg.Stop(locale.T("progress_found"))
	}

	parentCommitID := pj.CommitUUID()
	hasParentCommit := parentCommitID != ""

	pg = output.NewDotProgress(out, locale.T("progress_commit"), 10*time.Second)

	// Check if this is an addition or an update
	if params.Operation == model.OperationAdded && parentCommitID != "" {
		req, err := model.GetRequirement(parentCommitID, ns.String(), params.RequirementName)
		if err != nil {
			return errs.Wrap(err, "Could not get requirement")
		}
		if req != nil {
			params.Operation = model.OperationUpdated
		}
	}

	params.Analytics.EventWithLabel(
		anaConsts.CatPackageOp, fmt.Sprintf("%s-%s", params.Operation, langName), params.RequirementName,
	)

	if !hasParentCommit {
		languageFromNs := model.LanguageFromNamespace(ns.String())
		parentCommitID, err = model.CommitInitial(model.HostPlatform, languageFromNs, langVersion)
		if err != nil {
			return locale.WrapError(err, "err_install_no_project_commit", "Could not create initial commit for new project")
		}
	}

	var commitID strfmt.UUID
	commitID, err = model.CommitRequirement(parentCommitID, params.Operation, params.RequirementName, params.RequirementVersion, params.RequirementBitWidth, ns)
	if err != nil {
		if params.Operation == model.OperationRemoved && strings.Contains(err.Error(), "does not exist") {
			return locale.WrapInputError(err, "err_package_remove_does_not_exist", "Requirement is not installed: {{.V0}}", params.RequirementName)
		}
		return locale.WrapError(err, fmt.Sprintf("err_%s_%s", ns.Type(), params.Operation))
	}

	orderChanged := !hasParentCommit
	if hasParentCommit {
		revertCommit, err := model.GetRevertCommit(pj.CommitUUID(), commitID)
		if err != nil {
			return locale.WrapError(err, "err_revert_refresh")
		}
		orderChanged = len(revertCommit.Changeset) > 0
	}
	logging.Debug("Order changed: %v", orderChanged)

	pg.Stop(locale.T("progress_success"))

	var trigger target.Trigger
	switch ns.Type() {
	case model.NamespaceLanguage:
		trigger = target.TriggerLanguage
	case model.NamespacePackage, model.NamespaceBundle:
		trigger = target.TriggerPackage
	case model.NamespacePlatform:
		trigger = target.TriggerPlatform
	default:
		return errs.Wrap(err, "Unsupported namespace type: %s", ns.Type().String())
	}

	// refresh or install runtime
	err = runbits.RefreshRuntime(params.Auth, params.Output, params.Analytics, pj, storage.CachePath(), commitID, orderChanged, trigger, params.SvcModel)
	if err != nil {
		return err
	}

	if orderChanged {
		if err := pj.SetCommit(commitID.String()); err != nil {
			return locale.WrapError(err, "err_package_update_pjfile")
		}
	}

	// Print the result
	if !hasParentCommit {
		out.Print(locale.Tr("install_initial_success", pj.Source().Path()))
	}

	if params.RequirementVersion != "" {
		out.Print(locale.Tr(fmt.Sprintf("%s_version_%s", ns.Type(), params.Operation), params.RequirementName, params.RequirementVersion))
	} else {
		out.Print(locale.Tr(fmt.Sprintf("%s_%s", ns.Type(), params.Operation), params.RequirementName))
	}

	out.Print(locale.T("operation_success_local"))

	return nil
}

func supportedLanguageByName(supported []medmodel.SupportedLanguage, langName string) medmodel.SupportedLanguage {
	return funk.Find(supported, func(l medmodel.SupportedLanguage) bool { return l.Name == langName }).(medmodel.SupportedLanguage)
}

func resolvePkgAndNamespace(prompt prompt.Prompter, packageName string, nsType model.NamespaceType, supported []medmodel.SupportedLanguage) (string, model.Namespace, *medmodel.SupportedLanguage, error) {
	ns := model.NewBlankNamespace()

	// Find ingredients that match the input query
	ingredients, err := model.SearchIngredientsStrict(model.NewBlankNamespace(), packageName, false, false)
	if err != nil {
		return "", ns, nil, locale.WrapError(err, "err_pkgop_search_err", "Failed to check for ingredients.")
	}

	ingredients, err = model.FilterSupportedIngredients(supported, ingredients)
	if err != nil {
		return "", ns, nil, errs.Wrap(err, "Failed to filter out unsupported packages")
	}

	choices := []string{}
	values := map[string][]string{}
	for _, i := range ingredients {
		language := model.LanguageFromNamespace(*i.Ingredient.PrimaryNamespace)

		// Generate ingredient choices to present to the user
		name := fmt.Sprintf("%s (%s)", *i.Ingredient.Name, language)
		choices = append(choices, name)
		values[name] = []string{*i.Ingredient.Name, language}
	}

	if len(choices) == 0 {
		return "", ns, nil, locale.WrapInputError(err, "package_ingredient_alternatives_nolang", "", packageName)
	}

	// If we only have one ingredient match we're done; return it.
	if len(choices) == 1 {
		language := values[choices[0]][1]
		supportedLang := supportedLanguageByName(supported, language)
		return values[choices[0]][0], model.NewNamespacePkgOrBundle(language, nsType), &supportedLang, nil
	}

	// Prompt the user with the ingredient choices
	choice, err := prompt.Select(
		locale.Tl("prompt_pkgop_ingredient", "Multiple Matches"),
		locale.Tl("prompt_pkgop_ingredient_msg", "Your query has multiple matches, which one would you like to use?"),
		choices, &choices[0],
	)
	if err != nil {
		return "", ns, nil, locale.WrapError(err, "err_pkgop_select", "Need a selection.")
	}

	// Return the user selected ingredient
	language := values[choice][1]
	supportedLang := supportedLanguageByName(supported, language)
	return values[choice][0], model.NewNamespacePkgOrBundle(language, nsType), &supportedLang, nil
}

func getSuggestions(ns model.Namespace, name string) ([]string, error) {
	results, err := model.SearchIngredients(ns, name, false)
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

func initializeProject() (*project.Project, error) {
	target, err := os.Getwd()
	if err != nil {
		return nil, locale.WrapError(err, "err_add_get_wd", "Could not get working directory for new  project")
	}

	createParams := &projectfile.CreateParams{
		ProjectURL: constants.DashboardCommitURL,
		Directory:  target,
	}

	_, err = projectfile.Create(createParams)
	if err != nil {
		return nil, locale.WrapError(err, "err_add_create_projectfile", "Could not create new projectfile")
	}

	return project.FromPath(target)
}
