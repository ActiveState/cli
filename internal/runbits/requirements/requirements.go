package requirements

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/ActiveState/cli/internal/analytics"
	"github.com/go-openapi/strfmt"
	"github.com/thoas/go-funk"

	anaConsts "github.com/ActiveState/cli/internal/analytics/constants"
	"github.com/ActiveState/cli/internal/captain"
	"github.com/ActiveState/cli/internal/config"
	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/multilog"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/primer"
	"github.com/ActiveState/cli/internal/prompt"
	"github.com/ActiveState/cli/internal/runbits"
	"github.com/ActiveState/cli/internal/runbits/rtusage"
	medmodel "github.com/ActiveState/cli/pkg/platform/api/mediator/model"
	"github.com/ActiveState/cli/pkg/platform/authentication"
	"github.com/ActiveState/cli/pkg/platform/model"
	"github.com/ActiveState/cli/pkg/platform/runtime/artifact"
	"github.com/ActiveState/cli/pkg/platform/runtime/target"
	"github.com/ActiveState/cli/pkg/project"
	"github.com/ActiveState/cli/pkg/projectfile"
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

type RequirementOperation struct {
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

func (r *RequirementOperation) ExecuteRequirementOperation(requirementName, requirementVersion string, requirementBitWidth int, operation model.Operation, nsType model.NamespaceType) (rerr error) {
	var ns model.Namespace
	var langVersion string
	langName := "undetermined"

	out := r.Output
	var pg *output.Spinner
	defer func() {
		if pg != nil {
			// This is a bit awkward, but it would be even more awkward to manually address this for every error condition
			pg.Stop(locale.T("progress_fail"))
		}
	}()

	var err error
	pj := r.Project
	if pj == nil {
		pg = output.StartSpinner(out, locale.Tl("progress_project", "", requirementName), constants.TerminalAnimationInterval)
		pj, err = initializeProject()
		if err != nil {
			return locale.WrapError(err, "err_package_get_project", "Could not get project from path")
		}
		pg.Stop(locale.T("progress_success"))
		pg = nil // The defer above will redundantly call pg.Stop on success if we don't set this to nil

		defer func() {
			if rerr != nil && !errors.Is(err, artifact.CamelRuntimeBuilding) {
				if err := os.Remove(pj.Source().Path()); err != nil {
					multilog.Error("could not remove temporary project file: %s", errs.JoinMessage(err))
				}
			}
		}()
	}
	out.Notice(locale.Tl("operating_message", "", pj.NamespaceString(), pj.Dir()))

	switch nsType {
	case model.NamespacePackage, model.NamespaceBundle:
		language, err := model.LanguageByCommit(pj.CommitUUID())
		if err == nil {
			langName = language.Name
			ns = model.NewNamespacePkgOrBundle(langName, nsType)
		} else {
			logging.Debug("Could not get language from project: %v", err)
		}
	case model.NamespaceLanguage:
		ns = model.NewNamespaceLanguage()
	case model.NamespacePlatform:
		ns = model.NewNamespacePlatform()
	}

	rtusage.PrintRuntimeUsage(r.SvcModel, out, pj.Owner())

	var validatePkg = operation == model.OperationAdded && (ns.Type() == model.NamespacePackage || ns.Type() == model.NamespaceBundle)
	if !ns.IsValid() && (nsType == model.NamespacePackage || nsType == model.NamespaceBundle) {
		pg = output.StartSpinner(out, locale.Tl("progress_pkg_nolang", "", requirementName), constants.TerminalAnimationInterval)

		supported, err := model.FetchSupportedLanguages(model.HostPlatform)
		if err != nil {
			return errs.Wrap(err, "Failed to retrieve the list of supported languages")
		}

		var supportedLang *medmodel.SupportedLanguage
		requirementName, ns, supportedLang, err = resolvePkgAndNamespace(r.Prompt, requirementName, nsType, supported)
		if err != nil {
			return errs.Wrap(err, "Could not resolve pkg and namespace")
		}
		langVersion = supportedLang.DefaultVersion
		langName = supportedLang.Name

		validatePkg = false

		pg.Stop(locale.T("progress_found"))
		pg = nil
	}

	if strings.ToLower(requirementVersion) == latestVersion {
		requirementVersion = ""
	}

	if validatePkg {
		pg = output.StartSpinner(out, locale.Tl("progress_search", "", requirementName), constants.TerminalAnimationInterval)

		packages, err := model.SearchIngredientsStrict(ns, requirementName, false, false)
		if err != nil {
			return locale.WrapError(err, "package_err_cannot_obtain_search_results")
		}
		if len(packages) == 0 {
			suggestions, err := getSuggestions(ns, requirementName)
			if err != nil {
				multilog.Error("Failed to retrieve suggestions: %v", err)
			}
			if len(suggestions) == 0 {
				return locale.WrapInputError(err, "package_ingredient_alternatives_nosuggest", "", requirementName)
			}
			return locale.WrapInputError(err, "package_ingredient_alternatives", "", requirementName, strings.Join(suggestions, "\n"))
		}

		pg.Stop(locale.T("progress_found"))
		pg = nil
	}

	parentCommitID := pj.CommitUUID()
	hasParentCommit := parentCommitID != ""

	pg = output.StartSpinner(out, locale.T("progress_commit"), constants.TerminalAnimationInterval)

	// Check if this is an addition or an update
	if operation == model.OperationAdded && parentCommitID != "" {
		req, err := model.GetRequirement(parentCommitID, ns, requirementName)
		if err != nil {
			return errs.Wrap(err, "Could not get requirement")
		}
		if req != nil {
			operation = model.OperationUpdated
		}
	}

	r.Analytics.EventWithLabel(
		anaConsts.CatPackageOp, fmt.Sprintf("%s-%s", operation, langName), requirementName,
	)

	if !hasParentCommit {
		languageFromNs := model.LanguageFromNamespace(ns.String())
		parentCommitID, err = model.CommitInitial(model.HostPlatform, languageFromNs, langVersion)
		if err != nil {
			return locale.WrapError(err, "err_install_no_project_commit", "Could not create initial commit for new project")
		}
	}

	var commitID strfmt.UUID
	commitID, err = model.CommitRequirement(parentCommitID, operation, requirementName, requirementVersion, requirementBitWidth, ns)
	if err != nil {
		if operation == model.OperationRemoved && strings.Contains(err.Error(), "does not exist") {
			return locale.WrapInputError(err, "err_package_remove_does_not_exist", "Requirement is not installed: {{.V0}}", requirementName)
		}
		return locale.WrapError(err, fmt.Sprintf("err_%s_%s", ns.Type(), operation))
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
	pg = nil

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
	err = runbits.RefreshRuntime(r.Auth, r.Output, r.Analytics, pj, commitID, orderChanged, trigger, r.SvcModel)
	if err != nil {
		return errs.Wrap(err, "Failed to refresh runtime")
	}

	if orderChanged {
		if err := pj.SetCommit(commitID.String()); err != nil {
			return locale.WrapError(err, "err_package_update_pjfile")
		}
	}

	if !hasParentCommit {
		out.Notice(locale.Tr("install_initial_success", pj.Source().Path()))
	}

	// Print the result
	message := locale.Tr(fmt.Sprintf("%s_version_%s", ns.Type(), operation), requirementName, requirementVersion)
	if requirementVersion == "" {
		message = locale.Tr(fmt.Sprintf("%s_%s", ns.Type(), operation), requirementName)
	}
	out.Print(output.Prepare(
		message,
		&struct {
			Name      string `json:"name"`
			Version   string `json:"version,omitempty"`
			Type      string `json:"type"`
			Operation string `json:"operation"`
		}{
			requirementName,
			requirementVersion,
			ns.Type().String(),
			string(operation),
		}))

	out.Notice(locale.T("operation_success_local"))

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
