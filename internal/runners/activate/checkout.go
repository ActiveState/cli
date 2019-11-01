package activate

import (
	"path/filepath"

	"github.com/ActiveState/cli/internal/fileutils"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/pkg/cmdlets/git"
	"github.com/ActiveState/cli/pkg/platform/model"
	"github.com/ActiveState/cli/pkg/project"
	"github.com/ActiveState/cli/pkg/projectfile"
)

type CheckoutAble interface {
	Run(namespace string, path string) error
}

// Checkout will checkout the given platform project at the given path
// This includes cloning an associatd repository and creating the activestate.yaml
// It does not activate any environment
type Checkout struct {
	repo git.Repository
}

func NewCheckout(repo git.Repository) *Checkout {
	return &Checkout{repo}
}

func (r *Checkout) Run(namespace string, targetPath string) error {
	ns, fail := project.ParseNamespace(namespace)
	if fail != nil {
		return fail
	}

	pj, fail := model.FetchProjectByName(ns.Owner, ns.Project)
	if fail != nil {
		return fail
	}

	branch, fail := model.DefaultBranchForProject(pj)
	if fail != nil {
		return fail
	}

	// Clone the related repo, if it is defined
	if pj.RepoURL != nil {
		fail = r.repo.CloneProject(ns.Owner, ns.Project, targetPath)
		if fail != nil {
			return fail
		}
	}

	// Create the config file, if the repo clone didn't already create it
	configFile := filepath.Join(targetPath, constants.ConfigFileName)
	if !fileutils.FileExists(configFile) {
		fail = projectfile.Create(ns.Owner, ns.Project, branch.CommitID, targetPath)
		if fail != nil {
			return fail
		}
	}

	return nil
}
