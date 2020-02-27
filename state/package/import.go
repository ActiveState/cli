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

	rimport := reqsimport.Init()

	data, err := ioutil.ReadFile(ImportFlags.FileName)
	if err != nil {
		failures.Handle(err, locale.T("err_reading_file"))
		return
	}

	changeset, err := rimport.Changeset(data)
	if err != nil {
		failures.Handle(err, locale.T("err_obtaining_change_request"))
		return
	}

	msg := locale.T("commit_reqstext_message")

	if fail := model.CommitChangeset(proj.Owner(), proj.Name(), msg, changeset[1:]); fail != nil {
		failures.Handle(err, locale.T("err_"))
		return
	}
}
