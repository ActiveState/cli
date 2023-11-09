package projectmigration

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/multilog"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/prompt"
	"github.com/ActiveState/cli/pkg/localcommit"
	"github.com/ActiveState/cli/pkg/projectfile"
)

type projecter interface {
	Dir() string
	URL() string
	Path() string
	LegacyCommitID() string
}

var prompter prompt.Prompter
var out output.Outputer
var declined bool

// Register exists to avoid boilerplate in passing prompt and out to every caller of
// commitmediator.Get() for retrieving legacy commitId from activestate.yaml.
// This is an anti-pattern and is only used to make this legacy feature palatable.
func Register(prompter_ prompt.Prompter, out_ output.Outputer) {
	prompter = prompter_
	out = out_
}

func PromptAndMigrate(proj projecter) (bool, error) {
	if prompter == nil || out == nil {
		return false, errs.New("projectmigration.Register() has not been called")
	}

	if declined {
		return false, nil
	}

	if os.Getenv(constants.DisableProjectMigrationPrompt) == "true" {
		return false, nil
	}

	defaultChoice := false
	if migrate, err := prompter.Confirm("", locale.T("projectmigration_confirm"), &defaultChoice); err == nil && !migrate {
		if out.Config().Interactive {
			out.Notice(locale.Tl("projectmigration_declined", "Migration declined for now"))
		}
		declined = true
		return false, nil
	} else if err != nil {
		return false, errs.Wrap(err, "Could not confirm migration choice")
	}

	if err := localcommit.Set(proj.Dir(), proj.LegacyCommitID()); err != nil {
		return false, errs.Wrap(err, "Could not create local commit file")
	}

	for dir := proj.Dir(); filepath.Dir(dir) != dir; dir = filepath.Dir(dir) {
		if !fileutils.DirExists(filepath.Join(dir, ".git")) {
			continue
		}
		err := localcommit.AddToGitIgnore(dir)
		if err != nil {
			if !errors.Is(err, fs.ErrPermission) {
				multilog.Error("Unable to add local commit file to .gitignore: %v", err)
			}
			out.Notice(locale.T("notice_commit_id_gitignore"))
		}
		break
	}

	pf := projectfile.NewProjectField()
	if err := pf.LoadProject(proj.URL()); err != nil {
		return false, errs.Wrap(err, "Could not load activestate.yaml")
	}
	pf.StripCommitID()
	if err := pf.Save(proj.Path()); err != nil {
		return false, errs.Wrap(err, "Could not save activestate.yaml")
	}

	out.Notice(locale.Tl("projectmigration_success", "Your project was successfully migrated"))

	return true, nil
}
