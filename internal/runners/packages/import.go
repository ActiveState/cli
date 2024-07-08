package packages

import (
	"os"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/primer"
	"github.com/ActiveState/cli/internal/runbits/cves"
	"github.com/ActiveState/cli/internal/runbits/dependencies"
	"github.com/ActiveState/cli/internal/runbits/rationalize"
	"github.com/ActiveState/cli/internal/runbits/runtime"
	"github.com/ActiveState/cli/pkg/buildscript"
	"github.com/ActiveState/cli/pkg/localcommit"
	"github.com/ActiveState/cli/pkg/platform/api"
	"github.com/ActiveState/cli/pkg/platform/api/buildplanner/types"
	"github.com/ActiveState/cli/pkg/platform/api/reqsimport"
	"github.com/ActiveState/cli/pkg/platform/model"
	"github.com/ActiveState/cli/pkg/platform/model/buildplanner"
	"github.com/ActiveState/cli/pkg/platform/runtime/target"
)

const (
	defaultImportFile = "requirements.txt"
)

// Confirmer describes the behavior required to prompt a user for confirmation.
type Confirmer interface {
	Confirm(title, msg string, defaultOpt *bool) (bool, error)
}

// ChangesetProvider describes the behavior required to convert some file data
// into a changeset.
type ChangesetProvider interface {
	Changeset(contents []byte, lang string) (model.Changeset, error)
}

// ImportRunParams tracks the info required for running Import.
type ImportRunParams struct {
	FileName       string
	Language       string
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
func (i *Import) Run(params *ImportRunParams) error {
	logging.Debug("ExecuteImport")

	proj := i.prime.Project()
	if proj == nil {
		return rationalize.ErrNoProject
	}

	out := i.prime.Output()
	out.Notice(locale.Tr("operating_message", proj.NamespaceString(), proj.Dir()))

	if params.FileName == "" {
		params.FileName = defaultImportFile
	}

	latestCommit, err := localcommit.Get(proj.Dir())
	if err != nil {
		return locale.WrapError(err, "package_err_cannot_obtain_commit")
	}

	auth := i.prime.Auth()
	language, err := model.LanguageByCommit(latestCommit, auth)
	if err != nil {
		return locale.WrapError(err, "err_import_language", "Unable to get language from project")
	}

	pg := output.StartSpinner(out, locale.T("progress_commit"), constants.TerminalAnimationInterval)
	defer func() {
		if pg != nil {
			pg.Stop(locale.T("progress_fail"))
		}
	}()

	changeset, err := fetchImportChangeset(reqsimport.Init(), params.FileName, language.Name)
	if err != nil {
		return errs.Wrap(err, "Could not import changeset")
	}

	bp := buildplanner.NewBuildPlannerModel(auth)
	bs, err := bp.GetBuildScript(latestCommit.String())
	if err != nil {
		return locale.WrapError(err, "err_cannot_get_build_expression", "Could not get build expression")
	}

	if err := applyChangeset(changeset, bs); err != nil {
		return locale.WrapError(err, "err_cannot_apply_changeset", "Could not apply changeset")
	}

	msg := locale.T("commit_reqstext_message")
	commitID, err := bp.StageCommit(buildplanner.StageCommitParams{
		Owner:        proj.Owner(),
		Project:      proj.Name(),
		ParentCommit: latestCommit.String(),
		Description:  msg,
		Script:       bs,
	})
	if err != nil {
		return locale.WrapError(err, "err_commit_changeset", "Could not commit import changes")
	}

	pg.Stop(locale.T("progress_success"))
	pg = nil

	// Solve the runtime.
	rt, rtCommit, err := runtime.Solve(auth, out, i.prime.Analytics(), proj, &commitID, target.TriggerImport, i.prime.SvcModel(), i.prime.Config(), runtime.OptNone)
	if err != nil {
		return errs.Wrap(err, "Could not solve runtime")
	}

	// Output change summary.
	previousCommit, err := bp.FetchCommit(latestCommit, proj.Owner(), proj.Name(), nil)
	if err != nil {
		return errs.Wrap(err, "Failed to fetch build result for previous commit")
	}
	oldBuildPlan := previousCommit.BuildPlan()
	out.Notice("") // blank line
	dependencies.OutputChangeSummary(out, rtCommit.BuildPlan(), oldBuildPlan)

	// Report CVEs.
	if err := cves.NewCveReport(i.prime).Report(rtCommit.BuildPlan(), oldBuildPlan); err != nil {
		return errs.Wrap(err, "Could not report CVEs")
	}

	if err := localcommit.Set(proj.Dir(), commitID.String()); err != nil {
		return locale.WrapError(err, "err_package_update_commit_id")
	}

	// Update the runtime.
	if !i.prime.Config().GetBool(constants.AsyncRuntimeConfig) {
		out.Notice("")

		// refresh or install runtime
		err = runtime.UpdateByReference(rt, rtCommit, auth, proj, out, runtime.OptNone)
		if err != nil {
			return errs.Wrap(err, "Failed to update runtime")
		}
	}

	return nil
}

func fetchImportChangeset(cp ChangesetProvider, file string, lang string) (model.Changeset, error) {
	data, err := os.ReadFile(file)
	if err != nil {
		return nil, locale.WrapExternalError(err, "err_reading_changeset_file", "Cannot read import file: {{.V0}}", err.Error())
	}

	changeset, err := cp.Changeset(data, lang)
	if err != nil {
		return nil, locale.WrapError(err, "err_obtaining_change_request", "Could not process change set: {{.V0}}.", api.ErrorMessageFromPayload(err))
	}

	return changeset, err
}

func applyChangeset(changeset model.Changeset, bs *buildscript.BuildScript) error {
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

		req := types.Requirement{
			Name:      change.Requirement,
			Namespace: change.Namespace,
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
