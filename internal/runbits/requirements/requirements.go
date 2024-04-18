package requirements

import (
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/ActiveState/cli/internal/analytics"
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
	runbit "github.com/ActiveState/cli/internal/runbits/runtime"
	"github.com/ActiveState/cli/pkg/localcommit"
	"github.com/ActiveState/cli/pkg/platform/api/buildplanner/response"
	"github.com/ActiveState/cli/pkg/platform/api/buildplanner/types"
	medmodel "github.com/ActiveState/cli/pkg/platform/api/mediator/model"
	vulnModel "github.com/ActiveState/cli/pkg/platform/api/vulnerabilities/model"
	"github.com/ActiveState/cli/pkg/platform/api/vulnerabilities/request"
	"github.com/ActiveState/cli/pkg/platform/authentication"
	"github.com/ActiveState/cli/pkg/platform/model"
	"github.com/ActiveState/cli/pkg/platform/model/buildplanner"
	"github.com/ActiveState/cli/pkg/platform/runtime"
	"github.com/ActiveState/cli/pkg/platform/runtime/artifact"
	"github.com/ActiveState/cli/pkg/platform/runtime/buildplan"
	"github.com/ActiveState/cli/pkg/platform/runtime/buildscript"
	"github.com/ActiveState/cli/pkg/platform/runtime/setup/events"
	"github.com/ActiveState/cli/pkg/platform/runtime/target"
	"github.com/ActiveState/cli/pkg/project"
	"github.com/ActiveState/cli/pkg/sysinfo"
	"github.com/go-openapi/strfmt"
	"github.com/thoas/go-funk"
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

var errNoRequirements = errs.New("No requirements were provided")

var errInitialNoRequirement = errs.New("Could not find compatible requirement for initial commit")

var versionRe = regexp.MustCompile(`^\d(\.\d+)*$`)

// Requirement represents a package, language or platform requirement
// For now, be aware that you should never provide BOTH ns AND nsType, one or the other should always be nil, but never both.
// The refactor should clean this up.
type Requirement struct {
	Name          string
	Version       string
	Revision      int
	BitWidth      int // Only needed for platform requirements
	Namespace     *model.Namespace
	NamespaceType *model.NamespaceType
	Operation     types.Operation

	// The following fields are set during execution
	langName                string
	langVersion             string
	validatePkg             bool
	appendVersionWildcard   bool
	originalRequirementName string
	versionRequirements     []types.VersionRequirement
}

// ExecuteRequirementOperation executes the operation on the requirement
// This has become quite unwieldy, and is ripe for a refactor - https://activestatef.atlassian.net/browse/DX-1897
func (r *RequirementOperation) ExecuteRequirementOperation(ts *time.Time, requirements ...*Requirement) (rerr error) {
	defer r.rationalizeError(&rerr)

	if len(requirements) == 0 {
		return errNoRequirements
	}

	out := r.Output
	var pg *output.Spinner
	defer func() {
		if pg != nil {
			// This is a bit awkward, but it would be even more awkward to manually address this for every error condition
			pg.Stop(locale.T("progress_fail"))
		}
	}()

	if r.Project == nil {
		return rationalize.ErrNoProject
	}
	if r.Project.IsHeadless() {
		return rationalize.ErrHeadless
	}
	out.Notice(locale.Tr("operating_message", r.Project.NamespaceString(), r.Project.Dir()))

	if err := r.resolveNamespaces(ts, requirements...); err != nil {
		return errs.Wrap(err, "Could not resolve namespaces")
	}

	if err := r.validatePackages(requirements...); err != nil {
		return errs.Wrap(err, "Could not validate packages")
	}

	parentCommitID, err := localcommit.Get(r.Project.Dir())
	if err != nil {
		return errs.Wrap(err, "Unable to get local commit")
	}
	hasParentCommit := parentCommitID != ""

	pg = output.StartSpinner(out, locale.T("progress_commit"), constants.TerminalAnimationInterval)

	if err := r.checkForUpdate(parentCommitID, requirements...); err != nil {
		return locale.WrapError(err, "err_check_for_update", "Could not check for requirements updates")
	}

	if !hasParentCommit {
		// Use first requirement to extract language for initial commit
		var requirement *Requirement
		for _, r := range requirements {
			if r.Namespace.Type() == model.NamespacePackage || r.Namespace.Type() == model.NamespaceBundle {
				requirement = r
				break
			}
		}

		if requirement == nil {
			return errInitialNoRequirement
		}

		languageFromNs := model.LanguageFromNamespace(requirement.Namespace.String())
		parentCommitID, err = model.CommitInitial(sysinfo.OS().String(), languageFromNs, requirement.langVersion, r.Auth)
		if err != nil {
			return locale.WrapError(err, "err_install_no_project_commit", "Could not create initial commit for new project")
		}
	}

	if err := r.resolveRequirements(requirements...); err != nil {
		return locale.WrapError(err, "err_resolve_requirements", "Could not resolve one or more requirements")
	}

	var stageCommitReqs []buildplanner.StageCommitRequirement
	for _, requirement := range requirements {
		stageCommitReqs = append(stageCommitReqs, buildplanner.StageCommitRequirement{
			Name:      requirement.Name,
			Version:   requirement.versionRequirements,
			Revision:  ptr.To(requirement.Revision),
			Namespace: requirement.Namespace.String(),
			Operation: requirement.Operation,
		})
	}

	params := buildplanner.StageCommitParams{
		Owner:        r.Project.Owner(),
		Project:      r.Project.Name(),
		ParentCommit: string(parentCommitID),
		Description:  commitMessage(requirements...),
		Requirements: stageCommitReqs,
		TimeStamp:    ts,
	}

	bp := buildplanner.NewBuildPlannerModel(r.Auth)
	commitID, err := bp.StageCommit(params)
	if err != nil {
		return locale.WrapError(err, "err_package_save_and_build", "Error occurred while trying to create a commit")
	}

	pg.Stop(locale.T("progress_success"))
	pg = nil

	if strings.ToLower(os.Getenv(constants.DisableRuntime)) != "true" {
		// Solve runtime
		rt, commit, changedArtifacts, err := r.solve(commitID, requirements[0].Namespace)
		if err != nil {
			return errs.Wrap(err, "Could not solve runtime")
		}

		// Report CVEs
		if err := r.cveReport(*changedArtifacts, requirements...); err != nil {
			return errs.Wrap(err, "Could not report CVEs")
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
		err = runbit.UpdateByReference(rt, commit, r.Auth, r.Project, r.Output)
		if err != nil {
			if !runbits.IsBuildError(err) {
				// If the error is not a build error we want to retain the changes
				if err2 := r.updateCommitID(commitID); err2 != nil {
					return errs.Pack(err, locale.WrapError(err2, "err_package_update_commit_id"))
				}
			}
			return errs.Wrap(err, "Failed to refresh runtime")
		}

	}

	if err := r.updateCommitID(commitID); err != nil {
		return locale.WrapError(err, "err_package_update_commit_id")
	}

	if !hasParentCommit {
		out.Notice(locale.Tr("install_initial_success", r.Project.Source().Path()))
	}

	// Print the result
	r.outputResults(requirements...)

	out.Notice(locale.T("operation_success_local"))

	return nil
}

type ResolveNamespaceError struct {
	error
	Name string
}

func (r *RequirementOperation) resolveNamespaces(ts *time.Time, requirements ...*Requirement) error {
	for _, requirement := range requirements {
		if err := r.resolveNamespace(ts, requirement); err != nil {
			return &ResolveNamespaceError{
				err,
				requirement.Name,
			}
		}
	}
	return nil
}

func (r *RequirementOperation) resolveNamespace(ts *time.Time, requirement *Requirement) error {
	requirement.langName = "undetermined"

	if requirement.NamespaceType != nil {
		switch *requirement.NamespaceType {
		case model.NamespacePackage, model.NamespaceBundle:
			commitID, err := localcommit.Get(r.Project.Dir())
			if err != nil {
				return errs.Wrap(err, "Unable to get local commit")
			}

			language, err := model.LanguageByCommit(commitID, r.Auth)
			if err == nil {
				requirement.langName = language.Name
				requirement.Namespace = ptr.To(model.NewNamespacePkgOrBundle(requirement.langName, *requirement.NamespaceType))
			} else {
				logging.Debug("Could not get language from project: %v", err)
			}
		case model.NamespaceLanguage:
			requirement.Namespace = ptr.To(model.NewNamespaceLanguage())
		case model.NamespacePlatform:
			requirement.Namespace = ptr.To(model.NewNamespacePlatform())
		}
	}

	ns := requirement.Namespace
	nsType := requirement.NamespaceType
	requirement.validatePkg = requirement.Operation == types.OperationAdded && ns != nil && (ns.Type() == model.NamespacePackage || ns.Type() == model.NamespaceBundle || ns.Type() == model.NamespaceLanguage)
	if (ns == nil || !ns.IsValid()) && nsType != nil && (*nsType == model.NamespacePackage || *nsType == model.NamespaceBundle) {
		pg := output.StartSpinner(r.Output, locale.Tr("progress_pkg_nolang", requirement.Name), constants.TerminalAnimationInterval)

		supported, err := model.FetchSupportedLanguages(sysinfo.OS().String())
		if err != nil {
			return errs.Wrap(err, "Failed to retrieve the list of supported languages")
		}

		var nsv model.Namespace
		var supportedLang *medmodel.SupportedLanguage
		requirement.Name, nsv, supportedLang, err = resolvePkgAndNamespace(r.Prompt, requirement.Name, *requirement.NamespaceType, supported, ts, r.Auth)
		if err != nil {
			return errs.Wrap(err, "Could not resolve pkg and namespace")
		}
		requirement.Namespace = &nsv
		requirement.langVersion = supportedLang.DefaultVersion
		requirement.langName = supportedLang.Name

		requirement.validatePkg = false

		pg.Stop(locale.T("progress_found"))
	}

	if requirement.Namespace == nil {
		return locale.NewError("err_package_invalid_namespace_detected", "No valid namespace could be detected")
	}

	return nil
}

func (r *RequirementOperation) validatePackages(requirements ...*Requirement) error {
	var requirementsToValidate []*Requirement
	for _, requirement := range requirements {
		if !requirement.validatePkg {
			continue
		}
		requirementsToValidate = append(requirementsToValidate, requirement)
	}

	if len(requirementsToValidate) == 0 {
		return nil
	}

	pg := output.StartSpinner(r.Output, locale.Tr("progress_search", strings.Join(requirementNames(requirementsToValidate...), ", ")), constants.TerminalAnimationInterval)
	for _, requirement := range requirementsToValidate {
		if err := r.validatePackage(requirement); err != nil {
			return errs.Wrap(err, "Could not validate package")
		}
	}
	pg.Stop(locale.T("progress_found"))

	return nil
}

func (r *RequirementOperation) validatePackage(requirement *Requirement) error {
	if strings.ToLower(requirement.Version) == latestVersion {
		requirement.Version = ""
	}

	requirement.originalRequirementName = requirement.Name
	normalized, err := model.FetchNormalizedName(*requirement.Namespace, requirement.Name, r.Auth)
	if err != nil {
		multilog.Error("Failed to normalize '%s': %v", requirement.Name, err)
	}

	packages, err := model.SearchIngredientsStrict(requirement.Namespace.String(), normalized, false, false, nil, r.Auth) // ideally case-sensitive would be true (PB-4371)
	if err != nil {
		return locale.WrapError(err, "package_err_cannot_obtain_search_results")
	}

	if len(packages) == 0 {
		suggestions, err := getSuggestions(*requirement.Namespace, requirement.Name, r.Auth)
		if err != nil {
			multilog.Error("Failed to retrieve suggestions: %v", err)
		}

		if len(suggestions) == 0 {
			return &ErrNoMatches{
				locale.WrapInputError(err, "package_ingredient_alternatives_nosuggest", "", requirement.Name),
				requirement.Name, nil}
		}

		return &ErrNoMatches{
			locale.WrapInputError(err, "package_ingredient_alternatives", "", requirement.Name, strings.Join(suggestions, "\n")),
			requirement.Name, ptr.To(strings.Join(suggestions, "\n"))}
	}

	requirement.Name = normalized

	// If a bare version number was given, and if it is a partial version number (e.g. requests@2),
	// we'll want to ultimately append a '.x' suffix.
	if versionRe.MatchString(requirement.Version) {
		for _, knownVersion := range packages[0].Versions {
			if knownVersion.Version == requirement.Version {
				break
			} else if strings.HasPrefix(knownVersion.Version, requirement.Version) {
				requirement.appendVersionWildcard = true
			}
		}
	}

	return nil
}

func (r *RequirementOperation) checkForUpdate(parentCommitID strfmt.UUID, requirements ...*Requirement) error {
	for _, requirement := range requirements {
		// Check if this is an addition or an update
		if requirement.Operation == types.OperationAdded && parentCommitID != "" {
			req, err := model.GetRequirement(parentCommitID, *requirement.Namespace, requirement.Name, r.Auth)
			if err != nil {
				return errs.Wrap(err, "Could not get requirement")
			}
			if req != nil {
				requirement.Operation = types.OperationUpdated
			}
		}

		r.Analytics.EventWithLabel(
			anaConsts.CatPackageOp, fmt.Sprintf("%s-%s", requirement.Operation, requirement.langName), requirement.Name,
		)
	}

	return nil
}

func (r *RequirementOperation) resolveRequirements(requirements ...*Requirement) error {
	for _, requirement := range requirements {
		if err := r.resolveRequirement(requirement); err != nil {
			return errs.Wrap(err, "Could not resolve requirement")
		}
	}
	return nil
}

func (r *RequirementOperation) resolveRequirement(requirement *Requirement) error {
	var err error
	requirement.Name, requirement.Version, err = model.ResolveRequirementNameAndVersion(requirement.Name, requirement.Version, requirement.BitWidth, *requirement.Namespace, r.Auth)
	if err != nil {
		return errs.Wrap(err, "Could not resolve requirement name and version")
	}

	versionString := requirement.Version
	if requirement.appendVersionWildcard {
		versionString += ".x"
	}

	requirement.versionRequirements, err = buildplanner.VersionStringToRequirements(versionString)
	if err != nil {
		return errs.Wrap(err, "Could not process version string into requirements")
	}

	return nil
}

func (r *RequirementOperation) solve(commitID strfmt.UUID, ns *model.Namespace) (
	_ *runtime.Runtime, _ *response.Commit, _ *artifact.ArtifactChangeset, rerr error,
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
	commit, err := setup.Solve()
	if err != nil {
		return nil, nil, nil, errs.Wrap(err, "Solve failed")
	}

	// Get old buildplan
	// We can't use the local store here; because it might not exist (ie. integrationt test, user cleaned cache, ..),
	// but also there's no guarantee the old one is sequential to the current.
	oldCommit, err := model.GetCommit(commitID, r.Auth)
	if err != nil {
		return nil, nil, nil, errs.Wrap(err, "Could not get commit")
	}

	var oldBuildPlan *response.BuildResponse
	if oldCommit.ParentCommitID != "" {
		bp := buildplanner.NewBuildPlannerModel(r.Auth)
		oldCommit, err := bp.FetchCommit(oldCommit.ParentCommitID, rtTarget.Owner(), rtTarget.Name(), nil)
		if err != nil {
			return nil, nil, nil, errs.Wrap(err, "Failed to fetch build result")
		}
		oldBuildPlan = oldCommit.Build
	}

	changedArtifacts, err := buildplan.NewArtifactChangesetByBuildPlan(oldBuildPlan, commit.Build, false, false, r.Config, r.Auth)
	if err != nil {
		return nil, nil, nil, errs.Wrap(err, "Could not get changed artifacts")
	}

	return rt, commit, &changedArtifacts, nil
}

func (r *RequirementOperation) cveReport(artifactChangeset artifact.ArtifactChangeset, requirements ...*Requirement) error {
	if !r.Auth.Authenticated() {
		return nil
	}

	names := requirementNames(requirements...)
	pg := output.StartSpinner(r.Output, locale.T("progress_cve_search", strings.Join(names, ", ")), constants.TerminalAnimationInterval)

	var ingredients []*request.Ingredient
	for _, requirement := range requirements {
		if requirement.Operation == types.OperationRemoved {
			continue
		}

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
	}

	ingredientVulnerabilities, err := model.FetchVulnerabilitiesForIngredients(r.Auth, ingredients)
	if err != nil {
		return errs.Wrap(err, "Failed to retrieve vulnerabilities")
	}

	// No vulnerabilities, nothing further to do here
	if len(ingredientVulnerabilities) == 0 {
		logging.Debug("No vulnerabilities found for ingredients")
		pg.Stop(locale.T("progress_safe"))
		pg = nil
		return nil
	}

	pg.Stop(locale.T("progress_unsafe"))
	pg = nil

	vulnerabilities := model.CombineVulnerabilities(ingredientVulnerabilities, names...)
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

func (r *RequirementOperation) updateCommitID(commitID strfmt.UUID) error {
	if err := localcommit.Set(r.Project.Dir(), commitID.String()); err != nil {
		return locale.WrapError(err, "err_package_update_commit_id")
	}

	if r.Config.GetBool(constants.OptinBuildscriptsConfig) {
		bp := buildplanner.NewBuildPlannerModel(r.Auth)
		expr, atTime, err := bp.GetBuildExpressionAndTime(commitID.String())
		if err != nil {
			return errs.Wrap(err, "Could not get remote build expr and time")
		}

		err = buildscript.Update(r.Project, atTime, expr)
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

func (r *RequirementOperation) outputResults(requirements ...*Requirement) {
	for _, requirement := range requirements {
		r.outputResult(requirement)
	}
}

func (r *RequirementOperation) outputResult(requirement *Requirement) {
	// Print the result
	message := locale.Tr(fmt.Sprintf("%s_version_%s", requirement.Namespace.Type(), requirement.Operation), requirement.Name, requirement.Version)
	if requirement.Version == "" {
		message = locale.Tr(fmt.Sprintf("%s_%s", requirement.Namespace.Type(), requirement.Operation), requirement.Name)
	}

	r.Output.Print(output.Prepare(
		message,
		&struct {
			Name      string `json:"name"`
			Version   string `json:"version,omitempty"`
			Type      string `json:"type"`
			Operation string `json:"operation"`
		}{
			requirement.Name,
			requirement.Version,
			requirement.Namespace.Type().String(),
			requirement.Operation.String(),
		}))

	if requirement.originalRequirementName != requirement.Name {
		r.Output.Notice(locale.Tl("package_version_differs",
			"Note: the actual package name ({{.V0}}) is different from the requested package name ({{.V1}})",
			requirement.Name, requirement.originalRequirementName))
	}
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
	names := []string{}
	for _, requirement := range requirements {
		names = append(names, requirement.Name)
	}
	return names
}
