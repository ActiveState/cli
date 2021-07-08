package packages

import (
	"io/ioutil"

	"github.com/ActiveState/cli/internal/installation/storage"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/machineid"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/primer"
	"github.com/ActiveState/cli/internal/prompt"
	"github.com/ActiveState/cli/internal/runbits"
	"github.com/ActiveState/cli/pkg/platform/api"
	gqlModel "github.com/ActiveState/cli/pkg/platform/api/graphql/model"
	"github.com/ActiveState/cli/pkg/platform/api/reqsimport"
	"github.com/ActiveState/cli/pkg/platform/authentication"
	"github.com/ActiveState/cli/pkg/platform/model"
	"github.com/ActiveState/cli/pkg/project"
	"github.com/go-openapi/strfmt"
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
	FileName string
	Language string
	Force    bool
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
	proj *project.Project
	cfg  configurable
}

type primeable interface {
	primer.Outputer
	primer.Prompter
	primer.Projecter
	primer.Auther
	primer.Configurer
}

// NewImport prepares an importation execution context for use.
func NewImport(prime primeable) *Import {
	return &Import{
		prime.Auth(),
		prime.Output(),
		prime.Prompt(),
		prime.Project(),
		prime.Config(),
	}
}

// Run executes the import behavior.
func (i *Import) Run(params ImportRunParams) error {
	logging.Debug("ExecuteImport")

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
		return locale.WrapError(err, "err_obtaining_change_request", "Could not process change set: {{.V0}}.", api.ErrorMessageFromPayload(err))
	}

	packageReqs := model.FilterCheckpointPackages(reqs)
	if len(packageReqs) > 0 {
		force := params.Force
		err = removeRequirements(i.Prompter, i.proj, force, packageReqs)
		if err != nil {
			return locale.WrapError(err, "err_cannot_remove_existing")
		}
	}

	msg := locale.T("commit_reqstext_message")
	commitID, err := commitChangeset(i.proj, msg, changeset)
	if err != nil {
		return locale.WrapError(err, "err_commit_changeset", "Could not commit import changes")
	}

	return runbits.RefreshRuntime(i.auth, i.out, i.proj, storage.CachePath(), commitID, true)
}

func removeRequirements(conf Confirmer, project *project.Project, force bool, reqs []*gqlModel.Requirement) error {
	if !force {
		msg := locale.T("confirm_remove_existing_prompt")

		confirmed, err := conf.Confirm(locale.T("confirm"), msg, new(bool))
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
		return nil, err
	}

	changeset, err := cp.Changeset(data, lang)
	if err != nil {
		return nil, err
	}

	return changeset, err
}

func commitChangeset(project *project.Project, msg string, changeset model.Changeset) (strfmt.UUID, error) {
	commitID, err := model.CommitChangeset(project.CommitUUID(), msg, machineid.UniqID(), changeset)
	if err != nil {
		return "", locale.WrapError(err, "err_packages_removed")
	}

	if err := project.SetCommit(commitID.String()); err != nil {
		return "", locale.WrapError(err, "err_package_update_pjfile")
	}
	return commitID, nil
}
