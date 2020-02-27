package pkg

import (
	"io/ioutil"

	"github.com/spf13/cobra"

	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/pkg/cmdlets/commands"
	"github.com/ActiveState/cli/pkg/platform/api/reqsimport"
	"github.com/ActiveState/cli/pkg/platform/model"
	"github.com/ActiveState/cli/pkg/project"
)

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
}{
	defaultImportFile,
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
	},
	Run: ExecuteImport,
}

// ExecuteImport is executed with `state package import` is ran
func ExecuteImport(cmd *cobra.Command, allArgs []string) {
	logging.Debug("ExecuteImport")

	proj, fail := project.GetSafe()
	if fail != nil {
		failures.Handle(fail, locale.T("err_"))
		return
	}

	if ImportFlags.FileName == "" {
		ImportFlags.FileName = defaultImportFile
	}

	changeset, err := importChangeset(reqsimport.Init(), ImportFlags.FileName)
	if err != nil {
		failures.Handle(err, locale.T("err_obtaining_change_request"))
		return
	}

	msg := locale.T("commit_reqstext_message")

	fail = model.CommitChangeset(proj.Owner(), proj.Name(), msg, changeset)
	if fail != nil {
		failures.Handle(err, locale.T("err_"))
		return
	}
}

func importChangeset(cp ChangesetProvider, file string) (model.Changeset, error) {
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
