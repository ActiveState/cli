package packages

import (
	"io/ioutil"

	"github.com/ActiveState/cli/internal/analytics"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/keypairs"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/primer"
	"github.com/ActiveState/cli/internal/prompt"
	"github.com/ActiveState/cli/internal/runbits"
	"github.com/ActiveState/cli/internal/runbits/commitmediator"
	"github.com/ActiveState/cli/pkg/platform/api"
	gqlModel "github.com/ActiveState/cli/pkg/platform/api/graphql/model"
	"github.com/ActiveState/cli/pkg/platform/api/reqsimport"
	"github.com/ActiveState/cli/pkg/platform/authentication"
	"github.com/ActiveState/cli/pkg/platform/model"
	"github.com/ActiveState/cli/pkg/platform/runtime/target"
	"github.com/ActiveState/cli/pkg/project"
	"github.com/go-openapi/strfmt"
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

	i.out.Notice(locale.Tl("operating_message", "", i.proj.NamespaceString(), i.proj.Dir()))

	if params.FileName == "" {
		params.FileName = defaultImportFile
	}

	latestCommit, err := model.BranchCommitID(i.proj.Owner(), i.proj.Name(), i.proj.BranchName())
	if err != nil {
		return locale.WrapError(err, "package_err_cannot_obtain_commit")
	}

	reqs, err := fetchCheckpoint(latestCommit)
	if err != nil {
		return locale.WrapError(err, "package_err_cannot_fetch_checkpoint")
	}

	lang, err := model.CheckpointToLanguage(reqs)
	if err != nil {
		return locale.WrapInputError(err, "err_import_language", "Your project does not have a language associated with it, please add a language first.")
	}

	changeset, err := fetchImportChangeset(reqsimport.Init(), params.FileName, lang.Name)
	if err != nil {
		return errs.Wrap(err, "Could not import changeset")
	}

	packageReqs := model.FilterCheckpointNamespace(reqs, model.NamespacePackage, model.NamespaceBundle)
	if len(packageReqs) > 0 {
		err = removeRequirements(i.Prompter, i.proj, params, packageReqs)
		if err != nil {
			return locale.WrapError(err, "err_cannot_remove_existing")
		}
	}

	msg := locale.T("commit_reqstext_message")
	commitID, err := commitChangeset(i.proj, msg, changeset)
	if err != nil {
		return locale.WrapError(err, "err_commit_changeset", "Could not commit import changes")
	}

	return runbits.RefreshRuntime(i.auth, i.out, i.analytics, i.proj, commitID, true, target.TriggerImport, i.svcModel)
}

func removeRequirements(conf Confirmer, project *project.Project, params *ImportRunParams, reqs []*gqlModel.Requirement) error {
	if !params.NonInteractive {
		msg := locale.T("confirm_remove_existing_prompt")

		defaultChoice := params.NonInteractive
		confirmed, err := conf.Confirm(locale.T("confirm"), msg, &defaultChoice)
		if err != nil {
			return err
		}
		if !confirmed {
			return locale.NewInputError("err_action_was_not_confirmed", "Cancelled Import.")
		}
	}

	removal := model.ChangesetFromRequirements(model.OperationRemoved, reqs)
	msg := locale.T("commit_reqstext_remove_existing_message")
	_, err := commitChangeset(project, msg, removal)
	return err
}

func fetchImportChangeset(cp ChangesetProvider, file string, lang string) (model.Changeset, error) {
	data, err := ioutil.ReadFile(file)
	if err != nil {
		return nil, locale.WrapInputError(err, "err_reading_changeset_file", "Cannot read import file: {{.V0}}", err.Error())
	}

	changeset, err := cp.Changeset(data, lang)
	if err != nil {
		return nil, locale.WrapError(err, "err_obtaining_change_request", "Could not process change set: {{.V0}}.", api.ErrorMessageFromPayload(err))
	}

	return changeset, err
}

func commitChangeset(project *project.Project, msg string, changeset model.Changeset) (strfmt.UUID, error) {
	localCommitID, err := commitmediator.Get(project)
	if err != nil {
		return "", errs.Wrap(err, "Unable to get local commit")
	}
	commitID, err := model.CommitChangeset(localCommitID, msg, changeset)
	if err != nil {
		return "", errs.AddTips(locale.WrapError(err, "err_packages_removed"),
			locale.T("commit_failed_push_tip"),
			locale.T("commit_failed_pull_tip"))
	}

	if err := commitmediator.Set(project, commitID.String()); err != nil {
		return "", locale.WrapError(err, "err_package_update_commit_id")
	}
	return commitID, nil
}
