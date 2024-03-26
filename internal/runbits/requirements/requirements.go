package requirements

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/ActiveState/cli/internal/analytics"
	runbit "github.com/ActiveState/cli/internal/runbits/runtime"
	"github.com/ActiveState/cli/pkg/localcommit"
	"github.com/ActiveState/cli/pkg/platform/runtime"
	"github.com/ActiveState/cli/pkg/platform/runtime/artifact"
	"github.com/ActiveState/cli/pkg/platform/runtime/buildplan"
	"github.com/ActiveState/cli/pkg/platform/runtime/setup/events"
	"github.com/thoas/go-funk"

	anaConsts "github.com/ActiveState/cli/internal/analytics/constants"
	"github.com/ActiveState/cli/internal/captain"
	"github.com/ActiveState/cli/internal/config"
	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	configMediator "github.com/ActiveState/cli/internal/mediators/config"
	"github.com/ActiveState/cli/internal/multilog"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/primer"
	"github.com/ActiveState/cli/internal/prompt"
	"github.com/ActiveState/cli/internal/rtutils/ptr"
	"github.com/ActiveState/cli/internal/runbits"
	"github.com/ActiveState/cli/internal/runbits/rationalize"
	bpModel "github.com/ActiveState/cli/pkg/platform/api/buildplanner/model"
	medmodel "github.com/ActiveState/cli/pkg/platform/api/mediator/model"
	vulnModel "github.com/ActiveState/cli/pkg/platform/api/vulnerabilities/model"
	"github.com/ActiveState/cli/pkg/platform/api/vulnerabilities/request"
	"github.com/ActiveState/cli/pkg/platform/authentication"
	"github.com/ActiveState/cli/pkg/platform/model"
	"github.com/ActiveState/cli/pkg/platform/runtime/buildscript"
	"github.com/ActiveState/cli/pkg/platform/runtime/target"
	"github.com/ActiveState/cli/pkg/project"
	"github.com/go-openapi/strfmt"
)

func init() {
	configMediator.RegisterOption(constants.SecurityPromptConfig, configMediator.Bool, true)
	configMediator.RegisterOption(constants.SecurityPromptLevelConfig, configMediator.String, vulnModel.SeverityCritical)
}

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

var versionRe = regexp.MustCompile(`^\d(\.\d+)*$`)

// ExecuteRequirementOperation executes the operation on the requirement
// This has become quite unwieldy, and is ripe for a refactor - https://activestatef.atlassian.net/browse/DX-1897
// For now, be aware that you should never provide BOTH ns AND nsType, one or the other should always be nil, but never both.
// The refactor should clean this up.
func (r *RequirementOperation) ExecuteRequirementOperation(
	requirementName, requirementVersion string, requirementRevision *int,
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
			commitID, err := localcommit.Get(r.Project.Dir())
			if err != nil {
				return errs.Wrap(err, "Unable to get local commit")
			}

			language, err := model.LanguageByCommit(commitID, r.Auth)
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
		requirementName, nsv, supportedLang, err = resolvePkgAndNamespace(r.Prompt, requirementName, *nsType, supported, ts, r.Auth)
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
	appendVersionWildcard := false
	if validatePkg {
		pg = output.StartSpinner(out, locale.Tr("progress_search", requirementName), constants.TerminalAnimationInterval)

		normalized, err := model.FetchNormalizedName(*ns, requirementName, r.Auth)
		if err != nil {
			multilog.Error("Failed to normalize '%s': %v", requirementName, err)
		}

		packages, err := model.SearchIngredientsStrict(ns.String(), normalized, false, false, nil, r.Auth) // ideally case-sensitive would be true (PB-4371)
		if err != nil {
			return locale.WrapError(err, "package_err_cannot_obtain_search_results")
		}

		if len(packages) == 0 {
			suggestions, err := getSuggestions(*ns, requirementName, r.Auth)
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

		// If a bare version number was given, and if it is a partial version number (e.g. requests@2),
		// we'll want to ultimately append a '.x' suffix.
		if versionRe.MatchString(requirementVersion) {
			for _, knownVersion := range packages[0].Versions {
				if knownVersion.Version == requirementVersion {
					break
				} else if strings.HasPrefix(knownVersion.Version, requirementVersion) {
					appendVersionWildcard = true
				}
			}
		}

		pg.Stop(locale.T("progress_found"))
		pg = nil
	}

	parentCommitID, err := localcommit.Get(r.Project.Dir())
	if err != nil {
		return errs.Wrap(err, "Unable to get local commit")
	}
	hasParentCommit := parentCommitID != ""

	pg = output.StartSpinner(out, locale.T("progress_commit"), constants.TerminalAnimationInterval)

	// Check if this is an addition or an update
	if operation == bpModel.OperationAdded && parentCommitID != "" {
		req, err := model.GetRequirement(parentCommitID, *ns, requirementName, r.Auth)
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
		parentCommitID, err = model.CommitInitial(model.HostPlatform, languageFromNs, langVersion, r.Auth)
		if err != nil {
			return locale.WrapError(err, "err_install_no_project_commit", "Could not create initial commit for new project")
		}
	}

	name, version, err := model.ResolveRequirementNameAndVersion(requirementName, requirementVersion, requirementBitWidth, *ns, r.Auth)
	if err != nil {
		return errs.Wrap(err, "Could not resolve requirement name and version")
	}

	versionString := version
	if appendVersionWildcard {
		versionString += ".x"
	}
	requirements, err := model.VersionStringToRequirements(versionString)
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
		RequirementRevision:  requirementRevision,
		Operation:            operation,
		TimeStamp:            ts,
	}

	bp := model.NewBuildPlannerModel(r.Auth)
	commitID, err := bp.StageCommit(params)
	if err != nil {
		return locale.WrapError(err, "err_package_save_and_build", "Error occurred while trying to create a commit")
	}

	pg.Stop(locale.T("progress_success"))
	pg = nil

	// Solve runtime
	rt, buildResult, changedArtifacts, err := r.solve(commitID, ns)
	if err != nil {
		return errs.Wrap(err, "Could not solve runtime")
	}

	// Report CVE's
	if err := r.cveReport(requirementName, requirementVersion, *changedArtifacts, operation, ns); err != nil {
		return err
	}

	// Start runtime update UI
	out.Notice("")
	if !rt.HasCache() {
		out.Notice(output.Title(locale.T("install_runtime")))
		out.Notice(locale.T("install_runtime_info"))
	} else {
		out.Notice(output.Title(locale.T("update_runtime")))
		out.Notice(locale.T("update_runtime_info"))
	}

	// refresh or install runtime
	err = runbit.UpdateByReference(rt, buildResult, r.Auth, r.Output, r.Project, r.Config)
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

func (r *RequirementOperation) solve(commitID strfmt.UUID, ns *model.Namespace) (
	_ *runtime.Runtime, _ *model.BuildResult, _ *artifact.ArtifactChangeset, rerr error,
) {
	// Initialize runtime
	var trigger target.Trigger
	switch ns.Type() {
	case model.NamespaceLanguage:
		trigger = target.TriggerLanguage
	case model.NamespacePlatform:
		trigger = target.TriggerPlatform
	default:
		trigger = target.TriggerPackage
	}

	spinner := output.StartSpinner(r.Output, locale.T("progress_solve_preruntime"), constants.TerminalAnimationInterval)

	defer func() {
		if rerr != nil {
			spinner.Stop(locale.T("progress_fail"))
		} else {
			spinner.Stop(locale.T("progress_success"))
		}
	}()

	rtTarget := target.NewProjectTarget(r.Project, &commitID, trigger)
	rt, err := runtime.New(rtTarget, r.Analytics, r.SvcModel, r.Auth, r.Config, r.Output)
	if err != nil {
		return nil, nil, nil, locale.WrapError(err, "err_packages_update_runtime_init", "Could not initialize runtime.")
	}

	setup := rt.Setup(&events.VoidHandler{})
	buildResult, err := setup.Solve()
	if err != nil {
		return nil, nil, nil, errs.Wrap(err, "Solve failed")
	}

	// Get old buildplan
	// We can't use the local store here; because it might not exist (ie. integrationt test, user cleaned cache, ..),
	// but also there's no guarantee the old one is sequential to the current.
	commit, err := model.GetCommit(commitID, r.Auth)
	if err != nil {
		return nil, nil, nil, errs.Wrap(err, "Could not get commit")
	}

	var oldBuildPlan *bpModel.Build
	if commit.ParentCommitID != "" {
		bp := model.NewBuildPlannerModel(r.Auth)
		oldBuildResult, err := bp.FetchBuildResult(commit.ParentCommitID, rtTarget.Owner(), rtTarget.Name())
		if err != nil {
			return nil, nil, nil, errs.Wrap(err, "Failed to fetch build result")
		}
		oldBuildPlan = oldBuildResult.Build
	}

	changedArtifacts, err := buildplan.NewArtifactChangesetByBuildPlan(oldBuildPlan, buildResult.Build, false, false, r.Config, r.Auth)
	if err != nil {
		return nil, nil, nil, errs.Wrap(err, "Could not get changed artifacts")
	}

	return rt, buildResult, &changedArtifacts, nil
}

func (r *RequirementOperation) cveReport(requirementName, requirementVersion string, artifactChangeset artifact.ArtifactChangeset, operation bpModel.Operation, ns *model.Namespace) error {
	if !r.Auth.Authenticated() || operation == bpModel.OperationRemoved {
		return nil
	}

	reqNameAndVersion := requirementName
	if requirementVersion != "" {
		reqNameAndVersion = fmt.Sprintf("%s@%s", requirementName, requirementVersion)
	}
	pg := output.StartSpinner(r.Output, locale.Tr("progress_cve_search", reqNameAndVersion), constants.TerminalAnimationInterval)

	ingredients := []*request.Ingredient{}
	for _, artifact := range artifactChangeset.Added {
		ingredients = append(ingredients, &request.Ingredient{
			Namespace: artifact.Namespace,
			Name:      artifact.Name,
			Version:   *artifact.Version,
		})
	}
	for _, artifact := range artifactChangeset.Updated {
		if !artifact.IngredientChange {
			continue // For CVE reporting we only care about ingredient changes
		}
		ingredients = append(ingredients, &request.Ingredient{
			Namespace: artifact.To.Namespace,
			Name:      artifact.To.Name,
			Version:   *artifact.To.Version,
		})
	}

	ingredientVulnerabilities, err := model.FetchVulnerabilitiesForIngredients(r.Auth, ingredients)
	if err != nil {
		return errs.Wrap(err, "Failed to retrieve vulnerabilities")
	}

	// No vulnerabilities, nothing further to do here
	if len(ingredientVulnerabilities) == 0 {
		pg.Stop(locale.T("progress_safe"))
		pg = nil
		return nil
	}

	vulnerabilities := model.CombineVulnerabilities(ingredientVulnerabilities, requirementName)

	pg.Stop(locale.T("progress_unsafe"))
	pg = nil

	r.summarizeCVEs(r.Output, vulnerabilities)

	if r.shouldPromptForSecurity(vulnerabilities) {
		cont, err := r.promptForSecurity()
		if err != nil {
			return errs.Wrap(err, "Failed to prompt for security")
		}

		if !cont {
			if !r.Prompt.IsInteractive() {
				return errs.AddTips(
					locale.NewInputError("err_pkgop_security_prompt", "Operation aborted due to security prompt"),
					locale.Tl("more_info_prompt", "To disable security prompting run: [ACTIONABLE]state config set security.prompt.enabled false[/RESET]"),
				)
			}
			return locale.NewInputError("err_pkgop_security_prompt", "Operation aborted due to security prompt")
		}
	}

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
	if err := localcommit.Set(r.Project.Dir(), commitID.String()); err != nil {
		return locale.WrapError(err, "err_package_update_commit_id")
	}

	if r.Config.GetBool(constants.OptinBuildscriptsConfig) {
		bp := model.NewBuildPlannerModel(r.Auth)
		expr, atTime, err := bp.GetBuildExpressionAndTime(commitID.String())
		if err != nil {
			return errs.Wrap(err, "Could not get remote build expr and time")
		}

		err = buildscript.Update(r.Project, atTime, expr, r.Auth)
		if err != nil {
			return locale.WrapError(err, "err_update_build_script")
		}
	}

	return nil
}

func (r *RequirementOperation) shouldPromptForSecurity(vulnerabilities model.VulnerableIngredientsByLevels) bool {
	if !r.Config.GetBool(constants.SecurityPromptConfig) || vulnerabilities.Count == 0 {
		return false
	}

	promptLevel := r.Config.GetString(constants.SecurityPromptLevelConfig)

	logging.Debug("Prompt level: ", promptLevel)
	switch promptLevel {
	case vulnModel.SeverityCritical:
		return vulnerabilities.Critical.Count > 0
	case vulnModel.SeverityHigh:
		return vulnerabilities.Critical.Count > 0 ||
			vulnerabilities.High.Count > 0
	case vulnModel.SeverityMedium:
		return vulnerabilities.Critical.Count > 0 ||
			vulnerabilities.High.Count > 0 ||
			vulnerabilities.Medium.Count > 0
	case vulnModel.SeverityLow:
		return vulnerabilities.Critical.Count > 0 ||
			vulnerabilities.High.Count > 0 ||
			vulnerabilities.Medium.Count > 0 ||
			vulnerabilities.Low.Count > 0
	}

	return false
}

func (r *RequirementOperation) summarizeCVEs(out output.Outputer, vulnerabilities model.VulnerableIngredientsByLevels) {
	out.Print("")

	switch {
	case vulnerabilities.CountPrimary == 0:
		out.Print(locale.Tr("warning_vulnerable_indirectonly", strconv.Itoa(vulnerabilities.Count)))
	case vulnerabilities.CountPrimary == vulnerabilities.Count:
		out.Print(locale.Tr("warning_vulnerable_directonly", strconv.Itoa(vulnerabilities.Count)))
	default:
		out.Print(locale.Tr("warning_vulnerable", strconv.Itoa(vulnerabilities.CountPrimary), strconv.Itoa(vulnerabilities.Count-vulnerabilities.CountPrimary)))
	}

	printVulnerabilities := func(vulnerableIngredients model.VulnerableIngredientsByLevel, name, color string) {
		if vulnerableIngredients.Count > 0 {
			ings := []string{}
			for _, vulns := range vulnerableIngredients.Ingredients {
				prefix := ""
				if vulnerabilities.Count > vulnerabilities.CountPrimary {
					prefix = fmt.Sprintf("%s@%s: ", vulns.IngredientName, vulns.IngredientVersion)
				}
				ings = append(ings, fmt.Sprintf("%s[CYAN]%s[/RESET]", prefix, strings.Join(vulns.CVEIDs, ", ")))
			}
			out.Print(fmt.Sprintf(" • [%s]%d %s:[/RESET] %s", color, vulnerableIngredients.Count, name, strings.Join(ings, ", ")))
		}
	}

	printVulnerabilities(vulnerabilities.Critical, locale.Tl("cve_critical", "Critical"), "RED")
	printVulnerabilities(vulnerabilities.High, locale.Tl("cve_high", "High"), "ORANGE")
	printVulnerabilities(vulnerabilities.Medium, locale.Tl("cve_medium", "Medium"), "YELLOW")
	printVulnerabilities(vulnerabilities.Low, locale.Tl("cve_low", "Low"), "MAGENTA")

	out.Print("")
	out.Print(locale.T("more_info_vulnerabilities"))
}

func (r *RequirementOperation) promptForSecurity() (bool, error) {
	confirm, err := r.Prompt.Confirm("", locale.Tr("prompt_continue_pkg_operation"), ptr.To(false))
	if err != nil {
		return false, locale.WrapError(err, "err_pkgop_confirm", "Need a confirmation.")
	}

	return confirm, nil
}

func supportedLanguageByName(supported []medmodel.SupportedLanguage, langName string) medmodel.SupportedLanguage {
	return funk.Find(supported, func(l medmodel.SupportedLanguage) bool { return l.Name == langName }).(medmodel.SupportedLanguage)
}

func resolvePkgAndNamespace(prompt prompt.Prompter, packageName string, nsType model.NamespaceType, supported []medmodel.SupportedLanguage, ts *time.Time, auth *authentication.Auth) (string, model.Namespace, *medmodel.SupportedLanguage, error) {
	ns := model.NewBlankNamespace()

	// Find ingredients that match the input query
	ingredients, err := model.SearchIngredientsStrict("", packageName, false, false, ts, auth)
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

func getSuggestions(ns model.Namespace, name string, auth *authentication.Auth) ([]string, error) {
	results, err := model.SearchIngredients(ns.String(), name, false, nil, auth)
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
