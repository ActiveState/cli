package projectmigration

import (
	_ "embed"
	"errors"
	"io/fs"
	"path/filepath"

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
	Source() *projectfile.Project
	Dir() string
	URL() string
	Path() string
	LegacyCommitID() string
	StripLegacyCommitID() error
}

var prompter prompt.Prompter
var out output.Outputer

// Register exists to avoid boilerplate in passing prompt and out to every caller of
// commitmediator.Get() for retrieving legacy commitId from activestate.yaml.
// This is an anti-pattern and is only used to make this legacy feature palatable.
func Register(prompter_ prompt.Prompter, out_ output.Outputer) {
	prompter = prompter_
	out = out_
}

func Migrate(proj projecter) error {
	if err := localcommit.Set(proj.Dir(), proj.LegacyCommitID()); err != nil {
		return errs.Wrap(err, "Could not create local commit file")
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

	// Add comment to activestate.yaml explaining migration
	asB, err := fileutils.ReadFile(proj.Source().Path())
	if err != nil {
		return errs.Wrap(err, "Could not read activestate.yaml")
	}

	asB = append([]byte(locale.T("projectmigration_asyaml_comment")), asB...)
	if err := fileutils.WriteFile(proj.Source().Path(), asB); err != nil {
		return errs.Wrap(err, "Could not write to activestate.yaml")
	}
	return nil
}
