package git

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/ActiveState/cli/internal/condition"
	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/print"
	"github.com/ActiveState/cli/pkg/platform/model"
	"github.com/ActiveState/cli/pkg/projectfile"
	"gopkg.in/src-d/go-git.v4"
)

var (
	// FailTargetDirNotEmpty indicates that the target directory for cloning
	// this git repository is not empty
	FailTargetDirNotEmpty = failures.Type("git.fail.dirnotempty")

	// FailProjectURLMismatch indicates that the project url does not match
	// that of the URL in the cloned repository's activestate.yaml
	FailProjectURLMismatch = failures.Type("git.fail.projecturlmismatch")
)

// CloneProjectRepo will attempt to clone the associalted public git repository
// for the project identified by <owner>/<name> to the given directory
// TODO: Is this function doing too much? ie. Should it just clone and leave the rest
// to other methods?
func CloneProjectRepo(owner, name, path string) *failures.Failure {
	if condition.InTest() {
		return nil
	}

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
		// TODO: Inspect and clean RepoURL to ensure we can only
		// clone public projects (ie. use HTTPS)
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

	fail = moveFiles(tempDir, path)
	if fail != nil {
		return fail
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

	project, fail := projectfile.Parse(projectFilePath)
	if fail != nil {
		return fail
	}

	if !strings.Contains(project.Project, fmt.Sprintf("%s/%s", owner, name)) {
		return FailProjectURLMismatch.New(locale.T("error_git_project_url_mismatch"))
	}

	return nil
}

func moveFiles(src, dest string) *failures.Failure {
	if !fileutils.DirExists(dest) {
		err := os.MkdirAll(dest, 0755)
		if err != nil {
			return failures.FailUserInput.Wrap(err)
		}
	}

	return fileutils.MoveAllFiles(src, dest)
}
