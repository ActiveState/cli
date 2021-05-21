package git

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/src-d/go-git.v4"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/pkg/platform/model"
	"github.com/ActiveState/cli/pkg/project"
	"github.com/ActiveState/cli/pkg/projectfile"
)

// Repository is the interface used to represent a version control system repository
type Repository interface {
	CloneProject(owner, name, path string, out output.Outputer) error
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
func (r *Repo) CloneProject(owner, name, path string, out output.Outputer) error {
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
	_, err = git.PlainClone(tempDir, false, &git.CloneOptions{
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

	err = ensureCorrectRepo(owner, name, filepath.Join(tempDir, constants.ConfigFileName))
	if err != nil {
		return locale.WrapError(err, "err_git_ensure_repo", "The activestate.yaml in the cloned repository does not match the project you are activating.")
	}

	err = moveFiles(tempDir, path)
	if err != nil {
		return locale.WrapError(err, "err_git_move_files", "Could not move cloned files")
	}

	return nil
}

func ensureCorrectRepo(owner, name, projectFilePath string) error {
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
		return locale.NewError("err_git_project_url_mismatch", "Cloned project file does not match expected")
	}

	return nil
}

func moveFiles(src, dest string) error {
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
