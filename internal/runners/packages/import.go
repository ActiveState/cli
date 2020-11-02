package packages

import (
	"io/ioutil"

	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/primer"
	"github.com/ActiveState/cli/internal/prompt"
	"github.com/ActiveState/cli/pkg/cmdlets/auth"
	"github.com/ActiveState/cli/pkg/platform/api"
	"github.com/ActiveState/cli/pkg/platform/api/reqsimport"
	"github.com/ActiveState/cli/pkg/platform/model"
	"github.com/ActiveState/cli/pkg/project"
)

const (
	defaultImportFile = "requirements.txt"
)

// Confirmer describes the behavior required to prompt a user for confirmation.
type Confirmer interface {
	Confirm(title, msg string, defaultOpt bool) (bool, *failures.Failure)
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
	out output.Outputer
	prompt.Prompter
}

type primeable interface {
	primer.Outputer
	primer.Prompter
	primer.Projecter
	primer.Auther
}

// NewImport prepares an importation execution context for use.
func NewImport(prime primeable) *Import {
	return &Import{
		prime.Output(),
		prime.Prompt(),
	}
}

// Run executes the import behavior.
func (i *Import) Run(params ImportRunParams) error {
	logging.Debug("ExecuteImport")

	if params.FileName == "" {
		params.FileName = defaultImportFile
	}

	fail := auth.RequireAuthentication(locale.T("auth_required_activate"), i.out, i.Prompter)
	if fail != nil {
		return fail.WithDescription("err_activate_auth_required")
	}

	proj, fail := project.GetSafe()
	if fail != nil {
		return fail.WithDescription("err_project_unavailable")
	}

	latestCommit, fail := model.LatestCommitID(proj.Owner(), proj.Name())
	if fail != nil {
		return fail.WithDescription("package_err_cannot_obtain_commit")
	}

	reqs, fail := fetchCheckpoint(latestCommit)
	if fail != nil {
		return fail.WithDescription("package_err_cannot_fetch_checkpoint")
	}

	lang, fail := model.CheckpointToLanguage(reqs)
	if fail != nil {
		return locale.WrapInputError(fail, "err_import_language", "Your project does not have a language associated with it, please add a language first.")
	}

	changeset, err := fetchImportChangeset(reqsimport.Init(), params.FileName, lang.Name)
	if err != nil {
		return locale.WrapError(err, "err_obtaining_change_request", "Could not process change set: {{.V0}}.", api.ErrorMessageFromPayload(err))
	}

	packageReqs := model.FilterCheckpointPackages(reqs)
	if len(packageReqs) > 0 {
		force := params.Force
		fail = removeRequirements(prompt.New(), proj.Owner(), proj.Name(), force, packageReqs)
		if fail != nil {
			return fail.WithDescription("err_cannot_remove_existing")
		}
	}

	msg := locale.T("commit_reqstext_message")
	fail = model.CommitChangeset(proj.Owner(), proj.Name(), msg, changeset)
	if fail != nil {
		return fail.WithDescription("err_cannot_commit_changeset")
	}

	i.out.Notice(locale.T("update_config"))

	return nil
}

func removeRequirements(conf Confirmer, pjOwner, pjName string, force bool, reqs model.Checkpoint) *failures.Failure {
	if !force {
		msg := locale.T("confirm_remove_existing_prompt")

		confirmed, fail := conf.Confirm(locale.T("confirm"), msg, false)
		if fail != nil {
			return fail
		}
		if !confirmed {
			return failures.FailUserInput.New(locale.Tl("err_action_was_not_confirmed", "Cancelled Import."))
		}
	}

	removal := model.ChangesetFromRequirements(model.OperationRemoved, reqs)
	msg := locale.T("commit_reqstext_remove_existing_message")

	fail := model.CommitChangeset(pjOwner, pjName, msg, removal)
	if fail != nil {
		return fail.WithDescription("err_packages_removed")
	}

	return nil
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
