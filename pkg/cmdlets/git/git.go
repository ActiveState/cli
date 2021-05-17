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
		return err
	}

	tempDir, err := ioutil.TempDir("", fmt.Sprintf("state-activate-repo-%s-%s", owner, name))
	if err != nil {
		return errs.Wrap(err, "OS failure")
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
		err = errs.Wrap(err, "Cmd failure")
		tipMsg := locale.Tl(
			"err_tip_git_ssh-add",
			"SSH might be unable to authenticate. Running `ssh-add` before trying again may help.",
		)
		return errs.AddTips(err, tipMsg)
	}

	err = ensureCorrectRepo(owner, name, filepath.Join(tempDir, constants.ConfigFileName))
	if err != nil {
		return err
	}

	err = moveFiles(tempDir, path)
	if err != nil {
		return err
	}
	return nil
}

func ensureCorrectRepo(owner, name, projectFilePath string) error {
	_, err := os.Stat(projectFilePath)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return errs.Wrap(err, "OS failure")
	}

	projectFile, err := projectfile.Parse(projectFilePath)
	if err != nil {
		return err
	}

	proj, err := project.NewLegacy(projectFile)
	if err != nil {
		return err
	}

	if !(strings.ToLower(proj.Owner()) == strings.ToLower(owner)) || !(strings.ToLower(proj.Name()) == strings.ToLower(name)) {
		return locale.NewError("ProjectURLMismatch")
	}

	return nil
}

func moveFiles(src, dest string) error {
	err := verifyDestinationDirectory(dest)
	if err != nil {
		return err
	}

	return fileutils.MoveAllFilesCrossDisk(src, dest)
}

func verifyDestinationDirectory(dest string) error {
	if !fileutils.DirExists(dest) {
		return fileutils.Mkdir(dest)
	}

	empty, err := fileutils.IsEmptyDir(dest)
	if err != nil {
		return err
	}
	if !empty {
		return locale.NewError("TargetDirInUse")
	}

	return nil
}
