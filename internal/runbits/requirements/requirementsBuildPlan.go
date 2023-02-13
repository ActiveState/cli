package requirements

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	anaConsts "github.com/ActiveState/cli/internal/analytics/constants"
	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/multilog"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/runbits"
	"github.com/ActiveState/cli/internal/runbits/rtusage"
	bgModel "github.com/ActiveState/cli/pkg/platform/api/graphql/model/buildplanner"
	medmodel "github.com/ActiveState/cli/pkg/platform/api/mediator/model"
	"github.com/ActiveState/cli/pkg/platform/model"
	"github.com/ActiveState/cli/pkg/platform/runtime/artifact"
	bpModel "github.com/ActiveState/cli/pkg/platform/runtime/model"
	"github.com/ActiveState/cli/pkg/platform/runtime/target"
	"github.com/go-openapi/strfmt"
)

func (r *RequirementOperation) ExecuteRequirementOperationBuildPlan(requirementName string, requirementVersion string, requirementBitWidth int, operation bgModel.Operation, nsType model.NamespaceType) (rerr error) {
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

	var validatePkg = operation == bgModel.OperationAdd && (ns.Type() == model.NamespacePackage || ns.Type() == model.NamespaceBundle)
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
	if operation == bgModel.OperationAdd && parentCommitID != "" {
		req, err := model.GetRequirement(parentCommitID, ns, requirementName)
		if err != nil {
			return errs.Wrap(err, "Could not get requirement")
		}
		if req != nil {
			operation = bgModel.OperationUpdate
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

	bp := bpModel.NewBuildPlanner(r.Auth)
	commitID, err := bp.PushCommit(&bpModel.PushCommitParams{
		Owner:            pj.Owner(),
		Project:          pj.Name(),
		ParentCommit:     string(parentCommitID),
		BranchRef:        pj.BranchName(),
		Description:      fmt.Sprintf("%s-%s", operation, requirementName),
		PackageName:      requirementName,
		PackageVersion:   requirementVersion,
		PackageNamespace: ns,
		Operation:        operation,
		Time:             time.Now(),
	})
	if err != nil {
		return locale.WrapError(err, "err_package_save_and_build", "Could not save and build project")
	}

	orderChanged := !hasParentCommit
	if hasParentCommit {
		revertCommit, err := model.GetRevertCommit(pj.CommitUUID(), strfmt.UUID(commitID))
		if err != nil {
			return locale.WrapError(err, "err_revert_refresh")
		}
		orderChanged = len(revertCommit.Changeset) > 0
	}
	logging.Debug("Order changed: %v", orderChanged)

	pg.Stop(locale.T("progress_success"))
	pg = nil

	var trigger target.Trigger
	fmt.Println("Namespace type: ", ns.Type().String())
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
	err = runbits.RefreshRuntime(r.Auth, r.Output, r.Analytics, pj, strfmt.UUID(commitID), orderChanged, trigger, r.SvcModel)
	if err != nil {
		return err
	}

	if orderChanged {
		if err := pj.SetCommit(commitID); err != nil {
			return locale.WrapError(err, "err_package_update_pjfile")
		}
	}

	// Print the result
	if !hasParentCommit {
		out.Print(locale.Tr("install_initial_success", pj.Source().Path()))
	}

	if requirementVersion != "" {
		out.Print(locale.Tr(fmt.Sprintf("%s_version_%s", ns.Type(), operation), requirementName, requirementVersion))
	} else {
		out.Print(locale.Tr(fmt.Sprintf("%s_%s", ns.Type(), operation), requirementName))
	}

	out.Print(locale.T("operation_success_local"))

	return nil
}
