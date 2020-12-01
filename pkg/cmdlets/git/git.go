package git

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"gopkg.in/src-d/go-git.v4"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/pkg/platform/model"
	"github.com/ActiveState/cli/pkg/project"
	"github.com/ActiveState/cli/pkg/projectfile"
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
	project, fail := model.FetchProjectByName(owner, name)
	if fail != nil {
		return fail.ToError()
	}

	tempDir, err := ioutil.TempDir("", fmt.Sprintf("state-activate-repo-%s-%s", owner, name))
	if err != nil {
		return failures.FailOS.Wrap(err)
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
		return failures.FailCmd.Wrap(err).ToError()
	}

	fail = ensureCorrectRepo(owner, name, filepath.Join(tempDir, constants.ConfigFileName))
	if fail != nil {
		return fail.ToError()
	}

	fail = moveFiles(tempDir, path)
	if fail != nil {
		return fail.ToError()
	}
	return nil
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

	proj, fail := project.NewLegacy(projectFile)
	if fail != nil {
		return fail
	}

	if !(proj.Owner() == owner) || !(proj.Name() == name) {
		return FailProjectURLMismatch.New(locale.T("error_git_project_url_mismatch"))
	}

	return nil
}

func moveFiles(src, dest string) *failures.Failure {
	fail := verifyDestinationDirectory(dest)
	if fail != nil {
		return fail
	}

	return fileutils.MoveAllFilesCrossDisk(src, dest)
}

func verifyDestinationDirectory(dest string) *failures.Failure {
	if !fileutils.DirExists(dest) {
		return fileutils.Mkdir(dest)
	}

	empty, fail := fileutils.IsEmptyDir(dest)
	if fail != nil {
		return fail
	}
	if !empty {
		return FailTargetDirInUse.New(locale.T("error_git_target_dir_not_empty"))
	}

	return nil
}
