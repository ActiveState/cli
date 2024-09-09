package requirements

import (
	"errors"
	"fmt"
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
	"github.com/ActiveState/cli/internal/multilog"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/primer"
	"github.com/ActiveState/cli/internal/prompt"
	"github.com/ActiveState/cli/internal/rtutils/ptr"
	buildscript_runbit "github.com/ActiveState/cli/internal/runbits/buildscript"
	"github.com/ActiveState/cli/internal/runbits/cves"
	"github.com/ActiveState/cli/internal/runbits/dependencies"
	"github.com/ActiveState/cli/internal/runbits/rationalize"
	runtime_runbit "github.com/ActiveState/cli/internal/runbits/runtime"
	"github.com/ActiveState/cli/internal/runbits/runtime/trigger"
	"github.com/ActiveState/cli/pkg/buildscript"
	"github.com/ActiveState/cli/pkg/checkoutinfo"
	"github.com/ActiveState/cli/pkg/platform/api/buildplanner/response"
	"github.com/ActiveState/cli/pkg/platform/api/buildplanner/types"
	medmodel "github.com/ActiveState/cli/pkg/platform/api/mediator/model"
	"github.com/ActiveState/cli/pkg/platform/authentication"
	"github.com/ActiveState/cli/pkg/platform/model"
	bpModel "github.com/ActiveState/cli/pkg/platform/model/buildplanner"
	"github.com/ActiveState/cli/pkg/project"
	"github.com/ActiveState/cli/pkg/runtime"
	"github.com/ActiveState/cli/pkg/sysinfo"
	"github.com/go-openapi/strfmt"
	"github.com/thoas/go-funk"
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
// For now, be aware that you should never provide BOTH ns AND nsType, one or the other should always be nil, but never both.
// The refactor should clean this up.
type Requirement struct {
	Name          string
	Version       string
	Revision      *int
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

	parentCommitID, err := checkoutinfo.GetCommitID(r.Project.Dir())
	if err != nil {
		return errs.Wrap(err, "Unable to get local commit")
	}
	hasParentCommit := parentCommitID != ""

	pg = output.StartSpinner(r.Output, locale.T("progress_solve_preruntime"), constants.TerminalAnimationInterval)

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

	bp := bpModel.NewBuildPlannerModel(r.Auth)
	script, err := r.prepareBuildScript(bp, parentCommitID, requirements, ts)
	if err != nil {
		return errs.Wrap(err, "Could not prepare build script")
	}

	params := bpModel.StageCommitParams{
		Owner:        r.Project.Owner(),
		Project:      r.Project.Name(),
		ParentCommit: string(parentCommitID),
		Description:  commitMessage(requirements...),
		Script:       script,
	}

	// Solve runtime
	commit, err := bp.StageCommit(params)
	if err != nil {
		return errs.Wrap(err, "Could not stage commit")
	}

	ns := requirements[0].Namespace
	var trig trigger.Trigger
	switch ns.Type() {
	case model.NamespaceLanguage:
		trig = trigger.TriggerLanguage
	case model.NamespacePlatform:
		trig = trigger.TriggerPlatform
	default:
		trig = trigger.TriggerPackage
	}

	oldCommit, err := bp.FetchCommit(parentCommitID, r.Project.Owner(), r.Project.Name(), nil)
	if err != nil {
		return errs.Wrap(err, "Failed to fetch old build result")
	}

	pg.Stop(locale.T("progress_success"))
	pg = nil

	dependencies.OutputChangeSummary(r.prime.Output(), commit.BuildPlan(), oldCommit.BuildPlan())

	// Report CVEs
	if err := cves.NewCveReport(r.prime).Report(commit.BuildPlan(), oldCommit.BuildPlan()); err != nil {
		return errs.Wrap(err, "Could not report CVEs")
	}

	// Start runtime update UI
	if !r.Config.GetBool(constants.AsyncRuntimeConfig) {
		out.Notice("")

		// refresh or install runtime
		_, err = runtime_runbit.Update(r.prime, trig,
			runtime_runbit.WithCommit(commit),
			runtime_runbit.WithoutBuildscriptValidation(),
		)
		if err != nil {
			if !IsBuildError(err) {
				// If the error is not a build error we want to retain the changes
				if err2 := r.updateCommitID(commit.CommitID); err2 != nil {
					return errs.Pack(err, locale.WrapError(err2, "err_package_update_commit_id"))
				}
			}
			return errs.Wrap(err, "Failed to refresh runtime")
		}
	}

	if err := r.updateCommitID(commit.CommitID); err != nil {
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

func (r *RequirementOperation) prepareBuildScript(bp *bpModel.BuildPlanner, parentCommit strfmt.UUID, requirements []*Requirement, ts *time.Time) (*buildscript.BuildScript, error) {
	script, err := bp.GetBuildScript(r.Project.Owner(), r.Project.Name(), string(parentCommit))
	if err != nil {
		return nil, errs.Wrap(err, "Failed to get build expression")
	}

	if ts != nil {
		script.SetAtTime(*ts)
	} else {
		// If no atTime was provided then we need to ensure that the atTime in the script is updated to use
		// the most recent, which is either the current value or the platform latest.
		latest, err := model.FetchLatestTimeStamp(r.Auth)
		if err != nil {
			return nil, errs.Wrap(err, "Unable to fetch latest Platform timestamp")
		}
		if latest.After(script.AtTime()) {
			script.SetAtTime(latest)
		}
	}

	for _, req := range requirements {
		if req.Namespace.String() == types.NamespacePlatform {
			err = script.UpdatePlatform(req.Operation, strfmt.UUID(req.Name))
			if err != nil {
				return nil, errs.Wrap(err, "Failed to update build expression with platform")
			}
		} else {
			requirement := types.Requirement{
				Namespace:          req.Namespace.String(),
				Name:               req.Name,
				VersionRequirement: req.versionRequirements,
				Revision:           req.Revision,
			}

			err = script.UpdateRequirement(req.Operation, requirement)
			if err != nil {
				return nil, errs.Wrap(err, "Failed to update build expression with requirement")
			}
		}
	}

	return script, nil
}

type ResolveNamespaceError struct {
	Name string
}

func (e ResolveNamespaceError) Error() string {
	return "unable to resolve namespace"
}

func (r *RequirementOperation) resolveNamespaces(ts *time.Time, requirements ...*Requirement) error {
	for _, requirement := range requirements {
		if err := r.resolveNamespace(ts, requirement); err != nil {
			if err != errNoLanguage {
				err = errs.Pack(err, &ResolveNamespaceError{requirement.Name})
			}
			return errs.Wrap(err, "Unable to resolve namespace")
		}
	}
	return nil
}

func (r *RequirementOperation) resolveNamespace(ts *time.Time, requirement *Requirement) error {
	requirement.langName = "undetermined"

	if requirement.NamespaceType != nil {
		switch *requirement.NamespaceType {
		case model.NamespacePackage, model.NamespaceBundle:
			commitID, err := checkoutinfo.GetCommitID(r.Project.Dir())
			if err != nil {
				return errs.Wrap(err, "Unable to get local commit")
			}

			language, err := model.LanguageByCommit(commitID, r.Auth)
			if err != nil {
				logging.Debug("Could not get language from project: %v", err)
			}
			if language.Name == "" {
				return errNoLanguage
			}
			requirement.langName = language.Name
			requirement.Namespace = ptr.To(model.NewNamespacePkgOrBundle(requirement.langName, *requirement.NamespaceType))
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
				locale.WrapExternalError(err, "package_ingredient_alternatives_nosuggest", "", requirement.Name),
				requirement.Name, nil}
		}

		return &ErrNoMatches{
			locale.WrapExternalError(err, "package_ingredient_alternatives", "", requirement.Name, strings.Join(suggestions, "\n")),
			requirement.Name, ptr.To(strings.Join(suggestions, "\n"))}
	}

	if normalized != "" && normalized != requirement.Name {
		requirement.Name = normalized
	}

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

	requirement.versionRequirements, err = bpModel.VersionStringToRequirements(versionString)
	if err != nil {
		return errs.Wrap(err, "Could not process version string into requirements")
	}

	return nil
}

func (r *RequirementOperation) updateCommitID(commitID strfmt.UUID) error {
	if err := checkoutinfo.SetCommitID(r.Project.Dir(), commitID.String()); err != nil {
		return locale.WrapError(err, "err_package_update_commit_id")
	}

	if r.Config.GetBool(constants.OptinBuildscriptsConfig) {
		bp := bpModel.NewBuildPlannerModel(r.Auth)
		script, err := bp.GetBuildScript(r.Project.Owner(), r.Project.Name(), commitID.String())
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

	if requirement.originalRequirementName != requirement.Name && requirement.Operation != types.OperationRemoved {
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
		return "", ns, nil, locale.WrapExternalError(err, "package_ingredient_alternatives_nolang", "", packageName)
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
		locale.Tl("prompt_pkgop_ingredient_msg", "Your query has multiple matches. Which one would you like to use?"),
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
