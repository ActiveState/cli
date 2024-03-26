package requirements

import (
	"fmt"
	"strings"
	"time"

	anaConsts "github.com/ActiveState/cli/internal/analytics/constants"
	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/multilog"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/rtutils/ptr"
	"github.com/ActiveState/cli/internal/runbits/rationalize"
	runbit "github.com/ActiveState/cli/internal/runbits/runtime"
	"github.com/ActiveState/cli/pkg/localcommit"
	bpModel "github.com/ActiveState/cli/pkg/platform/api/buildplanner/model"
	medmodel "github.com/ActiveState/cli/pkg/platform/api/mediator/model"
	"github.com/ActiveState/cli/pkg/platform/model"
	"github.com/go-openapi/strfmt"
)

type Requirement struct {
	Name                    string
	Version                 string
	Revision                int
	BitWidth                int // Only needed for platform requirements
	Namespace               *model.Namespace
	NamespaceType           *model.NamespaceType
	Operation               bpModel.Operation
	langName                string
	langVersion             string
	validatePkg             bool
	appendVersionWildcard   bool
	originalRequirementName string
	versionRequirements     []bpModel.VersionRequirement
}

func (r *RequirementOperation) ExecuteRequirementOperationMultiple(ts *time.Time, requirements ...*Requirement) (rerr error) {
	if len(requirements) == 0 {
		return locale.NewError("err_no_requirements", "No requirements were provided")
	}

	defer r.rationalizeError(&rerr)

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
		return locale.WrapError(err, "err_resolve_namespaces", "Could not resolve one or more package namespaces")
	}

	if err := r.validatePackages(requirements...); err != nil {
		return locale.WrapError(err, "err_validate_packages", "Could not validate one or more packages")
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
		// Use first requirement to make initial commit
		requirement := requirements[0]
		languageFromNs := model.LanguageFromNamespace(requirement.Namespace.String())
		parentCommitID, err = model.CommitInitial(model.HostPlatform, languageFromNs, requirement.langVersion, r.Auth)
		if err != nil {
			return locale.WrapError(err, "err_install_no_project_commit", "Could not create initial commit for new project")
		}
	}

	if err := r.resolveRequirements(requirements...); err != nil {
		return locale.WrapError(err, "err_resolve_requirements", "Could not resolve one or more requirements")
	}

	var stageCommitReqs []model.StageCommitRequirement
	for _, requirement := range requirements {
		stageCommitReqs = append(stageCommitReqs, model.StageCommitRequirement{
			Name:      requirement.Name,
			Version:   requirement.versionRequirements,
			Revision:  ptr.To(requirement.Revision),
			Namespace: *requirement.Namespace,
			Operation: requirement.Operation,
		})
	}

	params := model.StageCommitParams{
		Owner:        r.Project.Owner(),
		Project:      r.Project.Name(),
		ParentCommit: string(parentCommitID),
		Description:  commitMessage(requirements...),
		Requirements: stageCommitReqs,
		TimeStamp:    ts,
	}

	bp := model.NewBuildPlannerModel(r.Auth)
	commitID, err := bp.StageCommit(params)
	if err != nil {
		return locale.WrapError(err, "err_package_save_and_build", "Error occurred while trying to create a commit")
	}

	pg.Stop(locale.T("progress_success"))
	pg = nil

	// Solve runtime
	rt, buildResult, changedArtifacts, err := r.solve(commitID, requirements[0].Namespace)
	if err != nil {
		return errs.Wrap(err, "Could not solve runtime")
	}

	// Report CVE's
	if err := r.cveReport(*changedArtifacts, requirements[0].Operation, requirements[0].Namespace); err != nil {
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

	// TODO: Print the result

	out.Notice(locale.T("operation_success_local"))

	return nil
}

func (r *RequirementOperation) resolveNamespaces(ts *time.Time, requirements ...*Requirement) error {
	for _, requirement := range requirements {
		if err := r.resolveNamespace(ts, requirement); err != nil {
			return errs.Wrap(err, "Could not resolve namespace")
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

	requirement.validatePkg = requirement.Operation == bpModel.OperationAdded && requirement.Namespace != nil && (requirement.Namespace.Type() == model.NamespacePackage || requirement.Namespace.Type() == model.NamespaceBundle)
	if (requirement.Namespace == nil || !requirement.Namespace.IsValid()) && requirement.NamespaceType != nil && (*requirement.NamespaceType == model.NamespacePackage || *requirement.NamespaceType == model.NamespaceBundle) {
		pg := output.StartSpinner(r.Output, locale.Tr("progress_pkg_nolang", requirement.Name), constants.TerminalAnimationInterval)

		supported, err := model.FetchSupportedLanguages(model.HostPlatform)
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
		pg = nil
	}

	if requirement.Namespace == nil {
		return locale.NewError("err_package_invalid_namespace_detected", "No valid namespace could be detected")
	}

	return nil
}

func (r *RequirementOperation) validatePackages(requirements ...*Requirement) error {
	for _, requirement := range requirements {
		if err := r.validatePackage(requirement); err != nil {
			return errs.Wrap(err, "Could not validate package")
		}
	}
	return nil
}

func (r *RequirementOperation) validatePackage(requirement *Requirement) error {
	if strings.ToLower(requirement.Version) == latestVersion {
		requirement.Version = ""
	}

	requirement.originalRequirementName = requirement.Name
	var pg *output.Spinner
	if requirement.validatePkg {
		pg = output.StartSpinner(r.Output, locale.Tr("progress_search", requirement.Name), constants.TerminalAnimationInterval)

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

		pg.Stop(locale.T("progress_found"))
		pg = nil
	}

	return nil
}

func (r *RequirementOperation) checkForUpdate(parentCommitID strfmt.UUID, requirements ...*Requirement) error {
	for _, requirement := range requirements {
		// Check if this is an addition or an update
		if requirement.Operation == bpModel.OperationAdded && parentCommitID != "" {
			req, err := model.GetRequirement(parentCommitID, *requirement.Namespace, requirement.Name, r.Auth)
			if err != nil {
				return errs.Wrap(err, "Could not get requirement")
			}
			if req != nil {
				requirement.Operation = bpModel.OperationUpdated
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

	requirement.versionRequirements, err = model.VersionStringToRequirements(versionString)
	if err != nil {
		return errs.Wrap(err, "Could not process version string into requirements")
	}

	return nil
}
