package git

import (
	"os"

	"github.com/ActiveState/cli/internal/condition"
	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/pkg/platform/model"
	"gopkg.in/src-d/go-git.v4"
)

// CloneProjectRepo will attempt to clone the associalted public git repository
// for the project identified by <owner>/<name>
func CloneProjectRepo(owner, name, path string) *failures.Failure {
	if condition.InTest() {
		return nil
	}

	projectModel, fail := model.FetchProjectByName(owner, name)
	if fail != nil {
		return fail
	}

	_, err := git.PlainClone(path, false, &git.CloneOptions{
		// TODO: Inspect and clean RepoURL to ensure we can only
		// clone public projects (ie. use HTTPS)
		URL:      projectModel.RepoURL.String(),
		Progress: os.Stdout,
	})
	if fail != nil {
		return failures.FailCmd.Wrap(err)
	}

	return nil
}
