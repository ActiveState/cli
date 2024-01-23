package requirements

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/ActiveState/cli/internal/analytics"
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
	"github.com/ActiveState/cli/internal/rtutils/ptr"
	"github.com/ActiveState/cli/internal/runbits"
	"github.com/ActiveState/cli/internal/runbits/commitmediator"
	"github.com/ActiveState/cli/internal/runbits/rationalize"
	bpModel "github.com/ActiveState/cli/pkg/platform/api/buildplanner/model"
	medmodel "github.com/ActiveState/cli/pkg/platform/api/mediator/model"
	"github.com/ActiveState/cli/pkg/platform/authentication"
	"github.com/ActiveState/cli/pkg/platform/model"
	"github.com/ActiveState/cli/pkg/platform/runtime/buildscript"
	"github.com/ActiveState/cli/pkg/platform/runtime/target"
	"github.com/ActiveState/cli/pkg/project"
	"github.com/go-openapi/strfmt"
)

type PackageVersion struct {
	captain.NameVersionValue
}

func (pv *PackageVersion) Set(arg string) error {
	err := pv.NameVersionValue.Set(arg)
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

type ErrNoMatches struct {
	*locale.LocalizedError
	Query        string
	Alternatives *string
}

// ExecuteRequirementOperation executes the operation on the requirement
// This has become quite unwieldy, and is ripe for a refactor - https://activestatef.atlassian.net/browse/DX-1897
// For now, be aware that you should never provide BOTH ns AND nsType, one or the other should always be nil, but never both.
// The refactor should clean this up.
func (r *RequirementOperation) ExecuteRequirementOperation(
	requirementName, requirementVersion string,
	requirementBitWidth int, // this is only needed for platform install/uninstall
	operation bpModel.Operation, ns *model.Namespace, nsType *model.NamespaceType, ts *time.Time) (rerr error) {
	defer r.rationalizeError(&rerr)

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
	if r.Project == nil {
		return rationalize.ErrNoProject
	}
	if r.Project.IsHeadless() {
		return rationalize.ErrHeadless
	}
	out.Notice(locale.Tr("operating_message", r.Project.NamespaceString(), r.Project.Dir()))

	if nsType != nil {
		switch *nsType {
		case model.NamespacePackage, model.NamespaceBundle:
			commitID, err := commitmediator.Get(r.Project)
			if err != nil {
				return errs.Wrap(err, "Unable to get local commit")
			}

			language, err := model.LanguageByCommit(commitID)
			if err == nil {
				langName = language.Name
				ns = ptr.To(model.NewNamespacePkgOrBundle(langName, *nsType))
			} else {
				logging.Debug("Could not get language from project: %v", err)
			}
		case model.NamespaceLanguage:
			ns = ptr.To(model.NewNamespaceLanguage())
		case model.NamespacePlatform:
			ns = ptr.To(model.NewNamespacePlatform())
		}
	}

	var validatePkg = operation == bpModel.OperationAdded && ns != nil && (ns.Type() == model.NamespacePackage || ns.Type() == model.NamespaceBundle)
	if (ns == nil || !ns.IsValid()) && nsType != nil && (*nsType == model.NamespacePackage || *nsType == model.NamespaceBundle) {
		pg = output.StartSpinner(out, locale.Tr("progress_pkg_nolang", requirementName), constants.TerminalAnimationInterval)

		supported, err := model.FetchSupportedLanguages(model.HostPlatform)
		if err != nil {
			return errs.Wrap(err, "Failed to retrieve the list of supported languages")
		}

		var nsv model.Namespace
		var supportedLang *medmodel.SupportedLanguage
		requirementName, nsv, supportedLang, err = resolvePkgAndNamespace(r.Prompt, requirementName, *nsType, supported, ts)
		if err != nil {
			return errs.Wrap(err, "Could not resolve pkg and namespace")
		}
		ns = &nsv
		langVersion = supportedLang.DefaultVersion
		langName = supportedLang.Name

		validatePkg = false

		pg.Stop(locale.T("progress_found"))
		pg = nil
	}

	if ns == nil {
		return locale.NewError("err_package_invalid_namespace_detected", "No valid namespace could be detected")
	}

	if strings.ToLower(requirementVersion) == latestVersion {
		requirementVersion = ""
	}

	origRequirementName := requirementName
	if validatePkg {
		pg = output.StartSpinner(out, locale.Tr("progress_search", requirementName), constants.TerminalAnimationInterval)

		normalized, err := model.FetchNormalizedName(*ns, requirementName)
		if err != nil {
			multilog.Error("Failed to normalize '%s': %v", requirementName, err)
		}

		packages, err := model.SearchIngredientsStrict(ns.String(), normalized, false, false, nil) // ideally case-sensitive would be true (PB-4371)
		if err != nil {
			return locale.WrapError(err, "package_err_cannot_obtain_search_results")
		}

		if len(packages) == 0 {
			suggestions, err := getSuggestions(*ns, requirementName)
			if err != nil {
				multilog.Error("Failed to retrieve suggestions: %v", err)
			}

			if len(suggestions) == 0 {
				return &ErrNoMatches{
					locale.WrapInputError(err, "package_ingredient_alternatives_nosuggest", "", requirementName),
					requirementName, nil}
			}

			return &ErrNoMatches{
				locale.WrapInputError(err, "package_ingredient_alternatives", "", requirementName, strings.Join(suggestions, "\n")),
				requirementName, ptr.To(strings.Join(suggestions, "\n"))}
		}

		requirementName = normalized

		pg.Stop(locale.T("progress_found"))
		pg = nil
	}

	parentCommitID, err := commitmediator.Get(r.Project)
	if err != nil {
		return errs.Wrap(err, "Unable to get local commit")
	}
	hasParentCommit := parentCommitID != ""

	pg = output.StartSpinner(out, locale.T("progress_commit"), constants.TerminalAnimationInterval)

	// Check if this is an addition or an update
	if operation == bpModel.OperationAdded && parentCommitID != "" {
		req, err := model.GetRequirement(parentCommitID, *ns, requirementName)
		if err != nil {
			return errs.Wrap(err, "Could not get requirement")
		}
		if req != nil {
			operation = bpModel.OperationUpdated
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

	name, version, err := model.ResolveRequirementNameAndVersion(requirementName, requirementVersion, requirementBitWidth, *ns)
	if err != nil {
		return errs.Wrap(err, "Could not resolve requirement name and version")
	}

	requirements, err := model.VersionStringToRequirements(version)
	if err != nil {
		return errs.Wrap(err, "Could not process version string into requirements")
	}

	params := model.StageCommitParams{
		Owner:                r.Project.Owner(),
		Project:              r.Project.Name(),
		ParentCommit:         string(parentCommitID),
		Description:          commitMessage(operation, name, version, *ns, requirementBitWidth),
		RequirementName:      name,
		RequirementVersion:   requirements,
		RequirementNamespace: *ns,
		Operation:            operation,
	}

	bp := model.NewBuildPlannerModel(r.Auth)
	commitID, err := bp.StageCommit(params)
	if err != nil {
		return locale.WrapError(err, "err_package_save_and_build", "Error occurred while trying to create a commit")
	}

	pg.Stop(locale.T("progress_success"))
	pg = nil

	// If applicable, output a list of additional dependencies (if any) being installed along with the
	// requested package or bundle.
	if operation == bpModel.OperationAdded &&
		ns != nil && (ns.Type() == model.NamespacePackage || ns.Type() == model.NamespaceBundle) &&
		hasParentCommit {
		err := r.outputAdditionalRequirements(parentCommitID, commitID, name)
		if err != nil {
			return errs.Wrap(err, "Unable to compute additional dependencies")
		}
	}

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
	err = runbits.RefreshRuntime(r.Auth, r.Output, r.Analytics, r.Project, commitID, true, trigger, r.SvcModel, r.Config)
	if err != nil {
		return r.handleRefreshError(err, parentCommitID)
	}

	if err := r.updateCommitID(commitID); err != nil {
		return locale.WrapError(err, "err_package_update_commit_id")
	}

	if !hasParentCommit {
		out.Notice(locale.Tr("install_initial_success", r.Project.Source().Path()))
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
			operation.String(),
		}))

	if origRequirementName != requirementName {
		out.Notice(locale.Tl("package_version_differs",
			"Note: the actual package name ({{.V0}}) is different from the requested package name ({{.V1}})",
			requirementName, origRequirementName))
	}

	out.Notice(locale.T("operation_success_local"))

	return nil
}

func (r *RequirementOperation) handleRefreshError(err error, parentCommitID strfmt.UUID) error {
	// If the error is a build error then return, if not update the commit ID then return
	if !runbits.IsBuildError(err) {
		if err := r.updateCommitID(parentCommitID); err != nil {
			return locale.WrapError(err, "err_package_update_commit_id")
		}
	}
	return err
}

func (r *RequirementOperation) updateCommitID(commitID strfmt.UUID) error {
	if err := commitmediator.Set(r.Project, commitID.String()); err != nil {
		return locale.WrapError(err, "err_package_update_commit_id")
	}

	bp := model.NewBuildPlannerModel(r.Auth)
	expr, err := bp.GetBuildExpression(r.Project.Owner(), r.Project.Name(), commitID.String())
	if err != nil {
		return errs.Wrap(err, "Could not get remote build expr")
	}

	// Note: a commit ID file needs to exist at this point.
	err = buildscript.Update(r.Project, expr, r.Auth)
	if err != nil {
		return locale.WrapError(err, "err_update_build_script")
	}

	return nil
}

func supportedLanguageByName(supported []medmodel.SupportedLanguage, langName string) medmodel.SupportedLanguage {
	return funk.Find(supported, func(l medmodel.SupportedLanguage) bool { return l.Name == langName }).(medmodel.SupportedLanguage)
}

func resolvePkgAndNamespace(prompt prompt.Prompter, packageName string, nsType model.NamespaceType, supported []medmodel.SupportedLanguage, ts *time.Time) (string, model.Namespace, *medmodel.SupportedLanguage, error) {
	ns := model.NewBlankNamespace()

	// Find ingredients that match the input query
	ingredients, err := model.SearchIngredientsStrict("", packageName, false, false, ts)
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
	results, err := model.SearchIngredients(ns.String(), name, false, nil)
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

func commitMessage(op bpModel.Operation, name, version string, namespace model.Namespace, word int) string {
	switch namespace.Type() {
	case model.NamespaceLanguage:
		return languageCommitMessage(op, name, version)
	case model.NamespacePlatform:
		return platformCommitMessage(op, name, version, word)
	case model.NamespacePackage, model.NamespaceBundle:
		return packageCommitMessage(op, name, version)
	}

	return ""
}

func languageCommitMessage(op bpModel.Operation, name, version string) string {
	var msgL10nKey string
	switch op {
	case bpModel.OperationAdded:
		msgL10nKey = "commit_message_added_language"
	case bpModel.OperationUpdated:
		msgL10nKey = "commit_message_updated_language"
	case bpModel.OperationRemoved:
		msgL10nKey = "commit_message_removed_language"
	}

	return locale.Tr(msgL10nKey, name, version)
}

func platformCommitMessage(op bpModel.Operation, name, version string, word int) string {
	var msgL10nKey string
	switch op {
	case bpModel.OperationAdded:
		msgL10nKey = "commit_message_added_platform"
	case bpModel.OperationUpdated:
		msgL10nKey = "commit_message_updated_platform"
	case bpModel.OperationRemoved:
		msgL10nKey = "commit_message_removed_platform"
	}

	return locale.Tr(msgL10nKey, name, strconv.Itoa(word), version)
}

func packageCommitMessage(op bpModel.Operation, name, version string) string {
	var msgL10nKey string
	switch op {
	case bpModel.OperationAdded:
		msgL10nKey = "commit_message_added_package"
	case bpModel.OperationUpdated:
		msgL10nKey = "commit_message_updated_package"
	case bpModel.OperationRemoved:
		msgL10nKey = "commit_message_removed_package"
	}

	if version == "" {
		version = locale.Tl("package_version_auto", "auto")
	}
	return locale.Tr(msgL10nKey, name, version)
}

// outputAdditionalRequirements computes and lists the additional dependencies being installed for the given package name.
// This should only be called if a package or bundle is being added. Otherwise, the results may be nonsensical.
func (r *RequirementOperation) outputAdditionalRequirements(parentCommitId, commitId strfmt.UUID, packageName string) (rerr error) {
	pg := output.StartSpinner(r.Output, locale.T("progress_dependencies"), constants.TerminalAnimationInterval)
	bp := model.NewBuildPlannerModel(r.Auth)

	// Fetch old build plan with sources to compare against.
	oldBuildPlan, err := bp.FetchBuildResult(parentCommitId, r.Project.Owner(), r.Project.Name())
	if err != nil {
		pg.Stop(locale.T("progress_fail"))
		return errs.Wrap(err, "Unable to fetch previous build plan to compare against")
	}
	resolvedRequirementsBefore := make(map[string]*bpModel.Source)
	for _, req := range oldBuildPlan.Build.Sources {
		resolvedRequirementsBefore[req.Name] = req
	}

	// Fetch new build plan with sources to compare with.
	newBuildPlan, err := bp.FetchBuildResult(commitId, r.Project.Owner(), r.Project.Name())
	if err != nil {
		pg.Stop(locale.T("progress_fail"))
		return errs.Wrap(err, "Unable to fetch new build plan to compare with")
	}

	// Perform the comparison and also get the resolved version number of the package being added.
	packageVersion := ""
	additions := make([]*bpModel.Source, 0)
	for _, req := range newBuildPlan.Build.Sources {
		if req.Name == packageName {
			packageVersion = req.Version
			continue
		}

		if !model.NamespaceMatch(req.Namespace, model.NamespaceBundlesMatch) &&
			!model.NamespaceMatch(req.Namespace, model.NamespacePackageMatch) {
			continue
		}

		if _, exists := resolvedRequirementsBefore[req.Name]; !exists {
			additions = append(additions, req)
		}
	}
	sort.SliceStable(additions, func(i, j int) bool {
		return additions[i].Name < additions[j].Name
	})

	pg.Stop(locale.T("progress_success"))
	if len(additions) == 0 {
		return nil
	}

	// List additional dependencies with indentation to match bullets.
	r.Output.Notice(locale.Tl(
		"changesummary_title",
		"   Installing [NOTICE]{{.V0}}@{{.V1}}[/RESET] will result in adding {{.V2}} additional dependencies.",
		packageName, packageVersion, strconv.Itoa(len(additions)),
	))
	for i, req := range additions {
		prefix := "   ├─"
		if i == len(additions)-1 {
			prefix = "   └─"
		}
		r.Output.Notice(fmt.Sprintf("  [DISABLED]%s[/RESET] %s@%s", prefix, req.Name, req.Version))
	}

	return nil
}
