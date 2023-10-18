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
	URL() string
	Path() string
	LegacyCommitID() string
}

type promptable interface {
	Confirm(title, message string, defaultChoice *bool) (bool, error)
}

// PromptToMigrateIfNecessary checks if the given project has been migrated to the new project
// format. If not, the user is prompted to do so. This should be called in main() as soon as a
// project file is parsed.
func PromptToMigrateIfNecessary(proj projecter, prompt promptable, out output.Outputer) error {
	projectDir := filepath.Dir(proj.Path())
	if _, err := localcommit.Get(projectDir); err == nil || !localcommit.IsFileDoesNotExistError(err) {
		return err
	}

	defaultChoice := false
	if migrate, err := prompt.Confirm("", locale.T("projectmigration_confirm"), &defaultChoice); err == nil && !migrate {
		if out.Config().Interactive {
			out.Notice(locale.Tl("projectmigration_declined", "Migration declined for now"))
		}
		return nil
	} else if err != nil {
		return locale.WrapError(err, "err_projectmigration_confirm", "Could not confirm migration choice")
	}

	if err := localcommit.Set(projectDir, proj.LegacyCommitID()); err != nil {
		return errs.Wrap(err, "Could not create local commit file")
	}

	if fileutils.DirExists(filepath.Join(projectDir, ".git")) {
		err := localcommit.AddToGitIgnore(projectDir)
		if err != nil {
			multilog.Error("Unable to add local commit file to .gitignore: %v", err)
			out.Notice(locale.T("notice_commit_id_gitignore"))
		}
	}

	pf := projectfile.NewProjectField()
	if err := pf.LoadProject(proj.URL()); err != nil {
		return errs.Wrap(err, "Could not load activestate.yaml")
	}
	pf.StripCommitID()
	if err := pf.Save(proj.Path()); err != nil {
		return errs.Wrap(err, "Could not save activestate.yaml")
	}

	out.Notice(locale.Tl("projectmigration_success", "Your project was successfully migrated"))

	return nil
}
