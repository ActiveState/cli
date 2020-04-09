package packages

import (
	"io/ioutil"

	"github.com/spf13/cobra"

	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/print"
	"github.com/ActiveState/cli/internal/prompt"
	"github.com/ActiveState/cli/pkg/cmdlets/auth"
	"github.com/ActiveState/cli/pkg/cmdlets/commands"
	"github.com/ActiveState/cli/pkg/platform/api/reqsimport"
	"github.com/ActiveState/cli/pkg/platform/model"
	"github.com/ActiveState/cli/pkg/project"
)

// Confirmer describes the behavior required to prompt a user for confirmation.
type Confirmer interface {
	Confirm(msg string, defaultOpt bool) (bool, *failures.Failure)
}

// ChangesetProvider describes the behavior required to convert some file data into a changeset.
type ChangesetProvider interface {
	Changeset([]byte) (model.Changeset, error)
}

const (
	defaultImportFile = "requirements.txt"
)

// ImportFlags holds the import-related flag values passed through the command line
var ImportFlags = struct {
	FileName string
	Force    bool
}{
	defaultImportFile,
	false,
}

// ImportCommand is the `package import` command struct
var ImportCommand = &commands.Command{
	Name:        "import",
	Description: "package_import_description",
	Flags: []*commands.Flag{
		{
			Name:        "file",
			Description: "package_import_flag_filename_description",
			Type:        commands.TypeString,
			StringVar:   &ImportFlags.FileName,
		},
		{
			Name:        "force",
			Description: "package_import_flag_force_description",
			Type:        commands.TypeBool,
			BoolVar:     &ImportFlags.Force,
		},
	},
	Run: ExecuteImport,
}

// ExecuteImport is executed with `state package import` is ran
func ExecuteImport(cmd *cobra.Command, allArgs []string) {
	logging.Debug("ExecuteImport")

	if ImportFlags.FileName == "" {
		ImportFlags.FileName = defaultImportFile
	}

	fail := auth.RequireAuthentication(locale.T("auth_required_activate"))
	if fail != nil {
		failures.Handle(fail, locale.T("err_activate_auth_required"))
		return
	}

	proj, fail := project.GetSafe()
	if fail != nil {
		failures.Handle(fail, locale.T("err_project_unavailable"))
		return
	}

	latestCommit, fail := model.LatestCommitID(proj.Owner(), proj.Name())
	if fail != nil {
		failures.Handle(fail, locale.T("package_err_cannot_obtain_commit"))
		return
	}

	reqs, fail := fetchCheckpoint(latestCommit)
	if fail != nil {
		failures.Handle(fail, locale.T("package_err_cannot_fetch_checkpoint"))
		return
	}

	changeset, err := fetchImportChangeset(reqsimport.Init(), ImportFlags.FileName)
	if err != nil {
		failures.Handle(err, locale.T("err_obtaining_change_request"))
		return
	}

	if len(reqs) > 0 {
		force := ImportFlags.Force
		fail = removeRequirements(prompt.New(), proj.Owner(), proj.Name(), force, reqs)
		if fail != nil {
			failures.Handle(fail, locale.T("err_cannot_remove_existing"))
			return
		}
	}

	msg := locale.T("commit_reqstext_message")
	fail = model.CommitChangeset(proj.Owner(), proj.Name(), msg, changeset)
	if fail != nil {
		failures.Handle(fail, locale.T("err_cannot_commit_changeset"))
		return
	}

	print.Warning(locale.T("package_update_config_file"))
}

func removeRequirements(conf Confirmer, pjOwner, pjName string, force bool, reqs model.Checkpoint) *failures.Failure {
	if !force {
		msg := locale.T("confirm_remove_existing_prompt")

		confirmed, fail := conf.Confirm(msg, false)
		if fail != nil {
			return fail
		}
		if !confirmed {
			return failures.FailUserInput.New("err_action_was_not_confirmed")
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

func fetchImportChangeset(cp ChangesetProvider, file string) (model.Changeset, error) {
	data, err := ioutil.ReadFile(ImportFlags.FileName)
	if err != nil {
		return nil, err
	}

	changeset, err := cp.Changeset(data)
	if err != nil {
		return nil, err
	}

	return changeset, err
}
