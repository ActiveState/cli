package git

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/print"
	"github.com/ActiveState/cli/pkg/platform/model"
	"github.com/ActiveState/cli/pkg/project"
	"github.com/ActiveState/cli/pkg/projectfile"
	"gopkg.in/src-d/go-git.v4"
)

var (
	// FailTargetDirInUse indicates that the target directory for cloning
	// this git repository is not empty
	FailTargetDirInUse = failures.Type("git.fail.dirinuse")

	// FailProjectURLMismatch indicates that the project url does not match
	// that of the URL in the cloned repository's activestate.yaml
	FailProjectURLMismatch = failures.Type("git.fail.projecturlmismatch")
)

// Repository is the interface used to represent a version control system repository
type Repository interface {
	CloneProject(owner, name, path string) *failures.Failure
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
func (r *Repo) CloneProject(owner, name, path string) *failures.Failure {
	project, fail := model.FetchProjectByName(owner, name)
	if fail != nil {
		return fail
	}

	tempDir, err := ioutil.TempDir("", fmt.Sprintf("state-activate-repo-%s-%s", owner, name))
	if err != nil {
		return failures.FailOS.Wrap(err)
	}
	defer os.RemoveAll(tempDir)

	print.Info(locale.Tr("git_cloning_project", owner, name))
	_, err = git.PlainClone(tempDir, false, &git.CloneOptions{
		URL:      project.RepoURL.String(),
		Progress: os.Stdout,
	})
	if err != nil {
		return failures.FailCmd.Wrap(err)
	}

	fail = ensureCorrectRepo(owner, name, filepath.Join(tempDir, constants.ConfigFileName))
	if fail != nil {
		return fail
	}

	return moveFiles(tempDir, path)
}

func ensureCorrectRepo(owner, name, projectFilePath string) *failures.Failure {
	_, err := os.Stat(projectFilePath)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return failures.FailOS.Wrap(err)
	}

	projectFile, fail := projectfile.Parse(projectFilePath)
	if fail != nil {
		return fail
	}

	proj, fail := project.New(projectFile)
	if fail != nil {
		return fail
	}

	if !(proj.Owner() == owner) || !(proj.Name() == name) {
		return FailProjectURLMismatch.New(locale.T("error_git_project_url_mismatch"))
	}

	return nil
}

func moveFiles(src, dest string) *failures.Failure {
	if fileutils.DirExists(dest) {
		return FailTargetDirInUse.New(locale.T("error_git_target_dir_exists"))
	}

	err := os.MkdirAll(dest, 0755)
	if err != nil {
		return failures.FailUserInput.Wrap(err)
	}

	return fileutils.MoveAllFiles(src, dest)
}
