package packages

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	anaConsts "github.com/ActiveState/cli/internal/analytics/constants"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/installation/storage"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/multilog"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/runbits"
	bgModel "github.com/ActiveState/cli/pkg/platform/api/graphql/model/buildplan"
	medmodel "github.com/ActiveState/cli/pkg/platform/api/mediator/model"
	"github.com/ActiveState/cli/pkg/platform/model"
	"github.com/ActiveState/cli/pkg/platform/runtime/artifact"
	bpModel "github.com/ActiveState/cli/pkg/platform/runtime/model"
	"github.com/ActiveState/cli/pkg/platform/runtime/target"
	"github.com/go-openapi/strfmt"
)

func executePackageOperationWithBuildPlan(prime primeable, packageName, packageVersion string, operation bgModel.Operation, nsType model.NamespaceType) (rerr error) {
	var ns model.Namespace
	var langVersion string
	langName := "undetermined"

	out := prime.Output()
	var pg *output.DotProgress
	defer func() {
		if pg != nil && !pg.Stopped() {
			pg.Stop(locale.T("progress_fail"))
		}
	}()

	var err error
	pj := prime.Project()
	if pj == nil {
		pg = output.NewDotProgress(out, locale.Tl("progress_project", "", packageName), 10*time.Second)
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
		language, err := model.LanguageByCommit(pj.CommitUUID())
		if err == nil {
			langName = language.Name
			ns = model.NewNamespacePkgOrBundle(langName, nsType)
		}
	}

	var validatePkg = operation == bgModel.OperationAdd
	if !ns.IsValid() {
		pg = output.NewDotProgress(out, locale.Tl("progress_pkg_nolang", "", packageName), 10*time.Second)

		supported, err := model.FetchSupportedLanguages(model.HostPlatform)
		if err != nil {
			return errs.Wrap(err, "Failed to retrieve the list of supported languages")
		}

		var supportedLang *medmodel.SupportedLanguage
		packageName, ns, supportedLang, err = resolvePkgAndNamespace(prime.Prompt(), packageName, nsType, supported)
		if err != nil {
			return errs.Wrap(err, "Could not resolve pkg and namespace")
		}
		langVersion = supportedLang.DefaultVersion
		langName = supportedLang.Name

		validatePkg = false

		pg.Stop(locale.T("progress_found"))
	}

	if strings.ToLower(packageVersion) == latestVersion {
		packageVersion = ""
	}

	if validatePkg {
		pg = output.NewDotProgress(out, locale.Tl("progress_search", "", packageName), 10*time.Second)

		packages, err := model.SearchIngredientsStrict(ns, packageName, false, false)
		if err != nil {
			return locale.WrapError(err, "package_err_cannot_obtain_search_results")
		}
		if len(packages) == 0 {
			suggestions, err := getSuggestions(ns, packageName)
			if err != nil {
				multilog.Error("Failed to retrieve suggestions: %v", err)
			}
			if len(suggestions) == 0 {
				return locale.WrapInputError(err, "package_ingredient_alternatives_nosuggest", "", packageName)
			}
			return locale.WrapInputError(err, "package_ingredient_alternatives", "", packageName, strings.Join(suggestions, "\n"))
		}

		pg.Stop(locale.T("progress_found"))
	}

	parentCommitID := pj.CommitUUID()
	hasParentCommit := parentCommitID != ""
	logging.Debug("Parent commit ID: %s", parentCommitID)
	logging.Debug("Has parent commit: %b", hasParentCommit)

	pg = output.NewDotProgress(out, locale.T("progress_commit"), 10*time.Second)

	// Check if this is an addition or an update
	// TODO: This will not work with the test harness
	// if operation == bgModel.OperationAdd && parentCommitID != "" {
	// 	req, err := model.GetRequirement(parentCommitID, ns.String(), packageName)
	// 	if err != nil {
	// 		return errs.Wrap(err, "Could not get requirement")
	// 	}
	// 	if req != nil {
	// 		operation = bgModel.OperationUpdate
	// 	}
	// }

	prime.Analytics().EventWithLabel(
		anaConsts.CatPackageOp, fmt.Sprintf("%s-%s", operation, langName), packageName,
	)

	if !hasParentCommit {
		languageFromNs := model.LanguageFromNamespace(ns.String())
		parentCommitID, err = model.CommitInitial(model.HostPlatform, languageFromNs, langVersion)
		if err != nil {
			return locale.WrapError(err, "err_install_no_project_commit", "Could not create initial commit for new project")
		}
	}

	bp := bpModel.NewBuildPlanner(prime.Auth())
	commitID, err := bp.SaveAndBuild(&bpModel.SaveAndBuildParams{
		Owner:            pj.Owner(),
		Project:          pj.Name(),
		ParentCommit:     string(parentCommitID),
		BranchRef:        pj.BranchName(),
		Description:      fmt.Sprintf("%s-%s", operation.String(), packageName),
		PackageName:      packageName,
		PackageVersion:   packageVersion,
		PackageNamespace: ns,
		Operation:        operation,
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

	// refresh or install runtime
	err = runbits.RefreshRuntime(prime.Auth(), prime.Output(), prime.Analytics(), pj, storage.CachePath(), strfmt.UUID(commitID), orderChanged, target.TriggerPackage, prime.SvcModel())
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

	if packageVersion != "" {
		out.Print(locale.Tr(fmt.Sprintf("%s_version_%s", ns.Type(), operation), packageName, packageVersion))
	} else {
		out.Print(locale.Tr(fmt.Sprintf("%s_%s", ns.Type(), operation), packageName))
	}

	out.Print(locale.T("operation_success_local"))

	return nil
}
