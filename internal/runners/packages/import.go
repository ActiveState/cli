package packages

import (
	"fmt"
	"os"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/primer"
	"github.com/ActiveState/cli/internal/runbits/cves"
	"github.com/ActiveState/cli/internal/runbits/dependencies"
	"github.com/ActiveState/cli/internal/runbits/org"
	"github.com/ActiveState/cli/internal/runbits/rationalize"
	runtime_runbit "github.com/ActiveState/cli/internal/runbits/runtime"
	"github.com/ActiveState/cli/internal/runbits/runtime/trigger"
	"github.com/ActiveState/cli/pkg/buildscript"
	"github.com/ActiveState/cli/pkg/localcommit"
	"github.com/ActiveState/cli/pkg/platform/api"
	"github.com/ActiveState/cli/pkg/platform/api/buildplanner/types"
	"github.com/ActiveState/cli/pkg/platform/api/reqsimport"
	"github.com/ActiveState/cli/pkg/platform/model"
	"github.com/ActiveState/cli/pkg/platform/model/buildplanner"
)

const (
	defaultImportFile = "requirements.txt"
)

// Confirmer describes the behavior required to prompt a user for confirmation.
type Confirmer interface {
	Confirm(title, msg string, defaultOpt *bool) (bool, error)
}

// ImportRunParams tracks the info required for running Import.
type ImportRunParams struct {
	FileName       string
	Language       string
	Namespace      string
	NonInteractive bool
}

// NewImportRunParams prepares the info required for running Import with default
// values.
func NewImportRunParams() *ImportRunParams {
	return &ImportRunParams{
		FileName: defaultImportFile,
	}
}

// Import manages the importing execution context.
type Import struct {
	prime primeable
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

// NewImport prepares an importation execution context for use.
func NewImport(prime primeable) *Import {
	return &Import{prime}
}

// Run executes the import behavior.
func (i *Import) Run(params *ImportRunParams) (rerr error) {
	defer rationalizeError(i.prime.Auth(), &rerr)
	logging.Debug("ExecuteImport")

	proj := i.prime.Project()
	if proj == nil {
		return rationalize.ErrNoProject
	}

	out := i.prime.Output()
	out.Notice(locale.Tr("operating_message", proj.NamespaceString(), proj.Dir()))

	filename := params.FileName
	if filename == "" {
		filename = defaultImportFile
	}

	localCommitId, err := localcommit.Get(proj.Dir())
	if err != nil {
		return locale.WrapError(err, "package_err_cannot_obtain_commit")
	}

	auth := i.prime.Auth()
	language, err := model.LanguageByCommit(localCommitId, auth)
	if err != nil {
		return locale.WrapError(err, "err_import_language", "Unable to get language from project")
	}

	pg := output.StartSpinner(i.prime.Output(), locale.T("progress_solve_preruntime"), constants.TerminalAnimationInterval)
	defer func() {
		if pg != nil {
			pg.Stop(locale.T("progress_fail"))
		}
	}()

	changeset, err := fetchImportChangeset(reqsimport.Init(), filename, language.Name, params.Namespace)
	if err != nil {
		return errs.Wrap(err, "Could not import changeset")
	}

	bp := buildplanner.NewBuildPlannerModel(auth, i.prime.SvcModel())
	bs, err := bp.GetBuildScript(localCommitId.String())
	if err != nil {
		return locale.WrapError(err, "err_cannot_get_build_expression", "Could not get build expression")
	}

	if err := i.applyChangeset(changeset, bs); err != nil {
		return locale.WrapError(err, "err_cannot_apply_changeset", "Could not apply changeset")
	}

	msg := locale.T("commit_reqstext_message")
	stagedCommit, err := bp.StageCommit(buildplanner.StageCommitParams{
		Owner:        proj.Owner(),
		Project:      proj.Name(),
		ParentCommit: localCommitId.String(),
		Description:  msg,
		Script:       bs,
	})
	// Always update the local commit ID even if the commit fails to build
	if stagedCommit != nil && stagedCommit.Commit != nil && stagedCommit.Commit.CommitID != "" {
		if err := localcommit.Set(proj.Dir(), stagedCommit.CommitID.String()); err != nil {
			return locale.WrapError(err, "err_package_update_commit_id")
		}
	}
	if err != nil {
		return locale.WrapError(err, "err_commit_changeset", "Could not commit import changes")
	}

	// Output change summary.
	previousCommit, err := bp.FetchCommit(localCommitId, proj.Owner(), proj.Name(), nil)
	if err != nil {
		return errs.Wrap(err, "Failed to fetch build result for previous commit")
	}

	pg.Stop(locale.T("progress_success"))
	pg = nil

	dependencies.OutputChangeSummary(i.prime.Output(), stagedCommit.BuildPlan(), previousCommit.BuildPlan())

	// Report CVEs.
	if err := cves.NewCveReport(i.prime).Report(stagedCommit.BuildPlan(), previousCommit.BuildPlan()); err != nil {
		return errs.Wrap(err, "Could not report CVEs")
	}

	out.Notice("") // blank line
	_, err = runtime_runbit.Update(i.prime, trigger.TriggerImport, runtime_runbit.WithCommitID(stagedCommit.CommitID))
	if err != nil {
		return errs.Wrap(err, "Runtime update failed")
	}

	out.Notice(locale.Tl("import_finished", "Import Finished"))

	return nil
}

func fetchImportChangeset(reqImport *reqsimport.ReqsImport, file string, lang string, namespace string) (model.Changeset, error) {
	data, err := os.ReadFile(file)
	if err != nil {
		return nil, locale.WrapExternalError(err, "err_reading_changeset_file", "Cannot read import file: {{.V0}}", err.Error())
	}

	changeset, err := reqImport.Changeset(data, lang, file, namespace)
	if err != nil {
		return nil, locale.WrapError(err, "err_obtaining_change_request", "Could not process change set: {{.V0}}.", api.ErrorMessageFromPayload(err))
	}

	return changeset, err
}

func (i *Import) applyChangeset(changeset model.Changeset, bs *buildscript.BuildScript) error {
	for _, change := range changeset {
		var expressionOperation types.Operation
		switch change.Operation {
		case string(model.OperationAdded):
			expressionOperation = types.OperationAdded
		case string(model.OperationRemoved):
			expressionOperation = types.OperationRemoved
		case string(model.OperationUpdated):
			expressionOperation = types.OperationUpdated
		}

		namespace := change.Namespace
		if namespace == "" {
			if !i.prime.Auth().Authenticated() {
				return rationalize.ErrNotAuthenticated
			}
			name, err := org.Get("", i.prime.Auth(), i.prime.Config())
			if err != nil {
				return errs.Wrap(err, "Unable to get an org for the user")
			}
			namespace = fmt.Sprintf("%s/%s", constants.PlatformPrivateNamespace, name)
		}

		req := types.Requirement{
			Name:      change.Requirement,
			Namespace: namespace,
		}

		for _, constraint := range change.VersionConstraints {
			req.VersionRequirement = append(req.VersionRequirement, types.VersionRequirement{
				types.VersionRequirementComparatorKey: constraint.Comparator,
				types.VersionRequirementVersionKey:    constraint.Version,
			})
		}

		if err := bs.UpdateRequirement(expressionOperation, req); err != nil {
			return errs.Wrap(err, "Could not update build expression")
		}
	}

	return nil
}
