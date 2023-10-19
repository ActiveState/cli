package projectmigration

import (
	"path/filepath"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/multilog"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/pkg/localcommit"
	"github.com/ActiveState/cli/pkg/projectfile"
)

type projecter interface {
	Dir() string
	URL() string
	Path() string
	LegacyCommitID() string
}

type promptable interface {
	Confirm(title, message string, defaultChoice *bool) (bool, error)
}

var declined bool

func PromptAndMigrate(proj projecter, prompt promptable, out output.Outputer) (bool, error) {
	if declined {
		return false, nil // already declined; do not prompt again
	}

	defaultChoice := false
	if migrate, err := prompt.Confirm("", locale.T("projectmigration_confirm"), &defaultChoice); err == nil && !migrate {
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

	if fileutils.DirExists(filepath.Join(proj.Dir(), ".git")) {
		err := localcommit.AddToGitIgnore(proj.Dir())
		if err != nil {
			multilog.Error("Unable to add local commit file to .gitignore: %v", err)
			out.Notice(locale.T("notice_commit_id_gitignore"))
		}
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
