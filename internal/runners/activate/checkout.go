package activate

import (
	"path/filepath"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/language"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/pkg/cmdlets/git"
	"github.com/ActiveState/cli/pkg/platform/model"
	"github.com/ActiveState/cli/pkg/project"
	"github.com/ActiveState/cli/pkg/projectfile"
)

// Checkout will checkout the given platform project at the given path
// This includes cloning an associated repository and creating the activestate.yaml
// It does not activate any environment
type Checkout struct {
	repo git.Repository
	output.Outputer
}

func NewCheckout(repo git.Repository, prime primeable) *Checkout {
	return &Checkout{repo, prime.Output()}
}

func (r *Checkout) Run(ns *project.Namespaced, branchName, targetPath string) error {
	if !ns.IsValid() {
		return locale.NewError("err_namespace_invalid", "Invalid namespace: {{.V0}}.", ns.String())
	}

	pj, err := model.FetchProjectByName(ns.Owner, ns.Project)
	if err != nil {
		return err
	}

	commitID := ns.CommitID
	if commitID == nil {
		branch, err := model.BranchForProjectByName(pj, branchName)
		if err != nil {
			return err
		}
		commitID = branch.CommitID
	}

	// Clone the related repo, if it is defined
	if pj.RepoURL != nil {
		err := r.repo.CloneProject(ns.Owner, ns.Project, targetPath, r.Outputer)
		if err != nil {
			return err
		}
	}

	language, err := getLanguage(ns.Owner, ns.Project, branchName)
	if err != nil {
		return err
	}

	// Create the config file, if the repo clone didn't already create it
	configFile := filepath.Join(targetPath, constants.ConfigFileName)
	if !fileutils.FileExists(configFile) {
		err = projectfile.Create(&projectfile.CreateParams{
			Owner:     ns.Owner,
			Project:   ns.Project,
			CommitID:  commitID,
			Directory: targetPath,
			Language:  language,
		})
		if err != nil {
			return err
		}
	}

	return nil
}

func getLanguage(owner, project, branch string) (string, error) {
	modelLanguage, err := model.DefaultLanguageForProject(owner, project, branch)
	if err != nil {
		return "", err
	}

	lang, err := language.MakeByNameAndVersion(modelLanguage.Name, modelLanguage.Version)
	if err != nil {
		return "", err
	}
	return lang.String(), nil
}
