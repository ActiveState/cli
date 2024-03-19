package packages

import (
	"os"

	"github.com/ActiveState/cli/internal/analytics"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/keypairs"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/primer"
	"github.com/ActiveState/cli/internal/prompt"
	"github.com/ActiveState/cli/internal/runbits"
	"github.com/ActiveState/cli/pkg/localcommit"
	"github.com/ActiveState/cli/pkg/platform/api"
	bpModel "github.com/ActiveState/cli/pkg/platform/api/buildplanner/model"
	"github.com/ActiveState/cli/pkg/platform/api/reqsimport"
	"github.com/ActiveState/cli/pkg/platform/authentication"
	"github.com/ActiveState/cli/pkg/platform/model"
	"github.com/ActiveState/cli/pkg/platform/runtime/buildexpression"
	"github.com/ActiveState/cli/pkg/platform/runtime/target"
	"github.com/ActiveState/cli/pkg/project"
)

const (
	defaultImportFile = "requirements.txt"
)

type configurable interface {
	keypairs.Configurable
}

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
	auth *authentication.Auth
	out  output.Outputer
	prompt.Prompter
	proj      *project.Project
	cfg       configurable
	analytics analytics.Dispatcher
	svcModel  *model.SvcModel
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
	return &Import{
		prime.Auth(),
		prime.Output(),
		prime.Prompt(),
		prime.Project(),
		prime.Config(),
		prime.Analytics(),
		prime.SvcModel(),
	}
}

// Run executes the import behavior.
func (i *Import) Run(params *ImportRunParams) error {
	logging.Debug("ExecuteImport")

	if i.proj == nil {
		return locale.NewInputError("err_no_project")
	}

	i.out.Notice(locale.Tr("operating_message", i.proj.NamespaceString(), i.proj.Dir()))

	if params.FileName == "" {
		params.FileName = defaultImportFile
	}

	latestCommit, err := localcommit.Get(i.proj.Dir())
	if err != nil {
		return locale.WrapError(err, "package_err_cannot_obtain_commit")
	}

	reqs, err := fetchCheckpoint(&latestCommit, i.auth)
	if err != nil {
		return locale.WrapError(err, "package_err_cannot_fetch_checkpoint")
	}

	lang, err := model.CheckpointToLanguage(reqs, i.auth)
	if err != nil {
		return locale.WrapInputError(err, "err_import_language", "Your project does not have a language associated with it, please add a language first.")
	}

	changeset, err := fetchImportChangeset(reqsimport.Init(), params.FileName, lang.Name)
	if err != nil {
		return errs.Wrap(err, "Could not import changeset")
	}

	bp := model.NewBuildPlannerModel(i.auth)
	be, err := bp.GetBuildExpression(latestCommit.String())
	if err != nil {
		return locale.WrapError(err, "err_cannot_get_build_expression", "Could not get build expression")
	}

	if err := applyChangeset(changeset, be); err != nil {
		return locale.WrapError(err, "err_cannot_apply_changeset", "Could not apply changeset")
	}

	if _, err := be.SetDefaultTimestamp(); err != nil {
		return locale.WrapError(err, "err_cannot_set_timestamp", "Could not set timestamp")
	}

	msg := locale.T("commit_reqstext_message")
	commitID, err := bp.StageCommit(model.StageCommitParams{
		Owner:        i.proj.Owner(),
		Project:      i.proj.Name(),
		ParentCommit: latestCommit.String(),
		Description:  msg,
		Expression:   be,
	})
	if err != nil {
		return locale.WrapError(err, "err_commit_changeset", "Could not commit import changes")
	}

	if err := localcommit.Set(i.proj.Dir(), commitID.String()); err != nil {
		return locale.WrapError(err, "err_package_update_commit_id")
	}

	return runbits.RefreshRuntime(i.auth, i.out, i.analytics, i.proj, commitID, true, target.TriggerImport, i.svcModel, i.cfg)
}

func fetchImportChangeset(cp ChangesetProvider, file string, lang string) (model.Changeset, error) {
	data, err := os.ReadFile(file)
	if err != nil {
		return nil, locale.WrapInputError(err, "err_reading_changeset_file", "Cannot read import file: {{.V0}}", err.Error())
	}

	changeset, err := cp.Changeset(data, lang)
	if err != nil {
		return nil, locale.WrapError(err, "err_obtaining_change_request", "Could not process change set: {{.V0}}.", api.ErrorMessageFromPayload(err))
	}

	return changeset, err
}

func applyChangeset(changeset model.Changeset, be *buildexpression.BuildExpression) error {
	for _, change := range changeset {
		var expressionOperation bpModel.Operation
		switch change.Operation {
		case string(model.OperationAdded):
			expressionOperation = bpModel.OperationAdded
		case string(model.OperationRemoved):
			expressionOperation = bpModel.OperationRemoved
		case string(model.OperationUpdated):
			expressionOperation = bpModel.OperationUpdated
		}

		req := bpModel.Requirement{
			Name:      change.Requirement,
			Namespace: change.Namespace,
		}

		for _, constraint := range change.VersionConstraints {
			req.VersionRequirement = append(req.VersionRequirement, bpModel.VersionRequirement{
				bpModel.VersionRequirementComparatorKey: constraint.Comparator,
				bpModel.VersionRequirementVersionKey:    constraint.Version,
			})
		}

		if err := be.UpdateRequirement(expressionOperation, req); err != nil {
			return errs.Wrap(err, "Could not update build expression")
		}
	}

	return nil
}
