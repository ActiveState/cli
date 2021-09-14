package activate

import (
	"path/filepath"

	"github.com/go-openapi/strfmt"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/errs"
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

func (r *Checkout) Run(ns *project.Namespaced, branchName, targetPath string) (bool, error) {
	var created bool
	if !ns.IsValid() {
		return created, locale.NewError("err_namespace_invalid", "Invalid namespace: {{.V0}}.", ns.String())
	}

	pj, err := model.FetchProjectByName(ns.Owner, ns.Project)
	if err != nil {
		return created, err
	}

	if branchName == "" {
		branch, err := model.DefaultBranchForProject(pj)
		if err != nil {
			return created, errs.Wrap(err, "Could not grab branch for project")
		}
		branchName = branch.Label
	}

	commitID := ns.CommitID
	if commitID == nil {
		branch, err := model.BranchForProjectByName(pj, branchName)
		if err != nil {
			return created, err
		}
		commitID = branch.CommitID
	}

	if commitID == nil {
		return created, errs.New("commitID is nil")
	}

	// Clone the related repo, if it is defined
	if pj.RepoURL != nil && *pj.RepoURL != "" {
		err := r.repo.CloneProject(ns.Owner, ns.Project, targetPath, r.Outputer)
		if err != nil {
			return created, err
		}
	}

	language, err := getLanguage(*commitID)
	if err != nil {
		return created, err
	}

	// Create the config file, if the repo clone didn't already create it
	configFile := filepath.Join(targetPath, constants.ConfigFileName)
	if !fileutils.FileExists(configFile) {
		_, err = projectfile.Create(&projectfile.CreateParams{
			Owner:      ns.Owner,
			Project:    ns.Project,
			CommitID:   commitID,
			BranchName: branchName,
			Directory:  targetPath,
			Language:   language,
		})
		if err != nil {
			return created, err
		}
		created = true
	}

	return created, nil
}

func getLanguage(commitID strfmt.UUID) (string, error) {
	modelLanguage, err := model.LanguageByCommit(commitID)
	if err != nil {
		return "", err
	}

	lang, err := language.MakeByNameAndVersion(modelLanguage.Name, modelLanguage.Version)
	if err != nil {
		return "", err
	}
	return lang.String(), nil
}
