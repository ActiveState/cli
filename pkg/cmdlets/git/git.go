package git

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/src-d/go-git.v4"

	"github.com/ActiveState/cli/internal/analytics"
	anaConsts "github.com/ActiveState/cli/internal/analytics/constants"
	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/pkg/platform/model"
	"github.com/ActiveState/cli/pkg/project"
	"github.com/ActiveState/cli/pkg/projectfile"
)

// Repository is the interface used to represent a version control system repository
type Repository interface {
	CloneProject(owner, name, path string, out output.Outputer, an analytics.AnalyticsDispatcher) error
}

// NewRepo returns a new repository
func NewRepo() *Repo {
	return &Repo{}
}

// Repo represents a git repository
type Repo struct {
}

// CloneProject will attempt to clone the associalted public git repository
// for the project identified by <owner>/<name> to the given directory
func (r *Repo) CloneProject(owner, name, path string, out output.Outputer, an analytics.AnalyticsDispatcher) error {
	project, err := model.FetchProjectByName(owner, name)
	if err != nil {
		return locale.WrapError(err, "err_git_fetch_project", "Could not fetch project details")
	}

	tempDir, err := ioutil.TempDir("", fmt.Sprintf("state-activate-repo-%s-%s", owner, name))
	if err != nil {
		return locale.WrapError(err, "err_git_tempdir", "Could not create temporary directory for git clone operation")
	}
	defer os.RemoveAll(tempDir)

	if project.RepoURL == nil {
		return locale.NewError("err_nil_repo_url", "Project returned empty repository URL")
	}

	out.Print(output.Heading(locale.Tr("git_cloning_project_heading")))
	out.Print(locale.Tr("git_cloning_project", *project.RepoURL))
	_, err = plainClone(tempDir, false, &git.CloneOptions{
		URL:      *project.RepoURL,
		Progress: os.Stdout,
	})
	if err != nil {
		err = locale.WrapError(err, "err_clone_repo", "Could not clone repository with URL: {{.V0}}, error received: {{.V1}}.", *project.RepoURL, err.Error())
		tipMsg := locale.Tl(
			"err_tip_git_ssh-add",
			"If you are using an SSH key please ensure it's configured by running `[ACTIONABLE]ssh-add <path-to-key>[/RESET]`.",
		)
		return errs.AddTips(err, tipMsg)
	}

	err = EnsureCorrectProject(owner, name, filepath.Join(tempDir, constants.ConfigFileName), *project.RepoURL, out, an)
	if err != nil {
		return locale.WrapError(err, "err_git_ensure_project", "Could not ensure that the activestate.yaml in the cloned repository matches the project you are activating.")
	}

	err = MoveFiles(tempDir, path)
	if err != nil {
		return locale.WrapError(err, "err_git_move_files", "Could not move cloned files")
	}

	return nil
}

// plainClone wraps git.PlainClone in order to handle a potential "wsarecv"
// error that is propagated via panic.
func plainClone(path string, isBare bool, o *git.CloneOptions) (r *git.Repository, derr error) {
	defer func() {
		if r := recover(); r != nil {
			if err, ok := r.(error); ok {
				derr = err
			} else {
				derr = fmt.Errorf("git.PlainClone recover type: %T, %v", r, r)
			}
			// removal tracked: https://www.pivotaltracker.com/story/show/179187192
			logging.Errorf("plain clone panic: %v", derr)
		}
	}()

	return git.PlainClone(path, isBare, o)
}

func EnsureCorrectProject(owner, name, projectFilePath, repoURL string, out output.Outputer, an analytics.AnalyticsDispatcher) error {
	if !fileutils.FileExists(projectFilePath) {
		return nil
	}

	projectFile, err := projectfile.Parse(projectFilePath)
	if err != nil {
		return locale.WrapError(err, "err_git_parse_projectfile", "Could not parse projectfile")
	}

	proj, err := project.NewLegacy(projectFile)
	if err != nil {
		return locale.WrapError(err, "err_git_project", "Could not create new project from project file at: {{.V0}}", projectFile.Path())
	}

	if !(strings.ToLower(proj.Owner()) == strings.ToLower(owner)) || !(strings.ToLower(proj.Name()) == strings.ToLower(name)) {
		out.Notice(locale.Tr("warning_git_project_mismatch", repoURL, project.NewNamespace(owner, name, "").String(), constants.DocumentationURLMismatch))
		err = proj.Source().SetNamespace(owner, name)
		if err != nil {
			return locale.WrapError(err, "err_git_update_mismatch", "Could not update projectfile namespace")
		}
		an.Event(anaConsts.CatMisc, "git-project-mismatch")
	}

	return nil
}

func MoveFiles(src, dest string) error {
	err := verifyDestinationDirectory(dest)
	if err != nil {
		return locale.WrapError(err, "err_git_verify_dir", "Could not verify destination directory")
	}

	err = fileutils.MoveAllFilesCrossDisk(src, dest)
	if err != nil {
		return locale.WrapError(err, "err_git_move_file", "Could not move files from {{.V0}} to {{.V1}}", src, dest)
	}

	return nil
}

func verifyDestinationDirectory(dest string) error {
	if !fileutils.DirExists(dest) {
		return fileutils.Mkdir(dest)
	}

	empty, err := fileutils.IsEmptyDir(dest)
	if err != nil {
		return locale.WrapError(err, "err_git_empty_dir", "Could not verify if destination directory is empty")
	}
	if !empty {
		return locale.NewError("err_git_in_use", "Destination directory is not empty")
	}

	return nil
}
