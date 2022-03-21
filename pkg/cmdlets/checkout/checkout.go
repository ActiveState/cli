package checkout

import (
	"path/filepath"

	"github.com/ActiveState/cli/internal/analytics"
	"github.com/ActiveState/cli/internal/config"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/primer"
	"github.com/go-openapi/strfmt"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/language"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/pkg/cmdlets/git"
	"github.com/ActiveState/cli/pkg/platform/model"
	"github.com/ActiveState/cli/pkg/project"
	"github.com/ActiveState/cli/pkg/projectfile"
)

type primeable interface {
	primer.Outputer
	primer.Analyticer
	primer.Configurer
}

// Checkout will checkout the given platform project at the given path
// This includes cloning an associated repository and creating the activestate.yaml
// It does not activate any environment
type Checkout struct {
	repo git.Repository
	output.Outputer
	config    *config.Instance
	analytics analytics.Dispatcher
}

func New(repo git.Repository, prime primeable) *Checkout {
	return &Checkout{repo, prime.Output(), prime.Config(), prime.Analytics()}
}

func (r *Checkout) Run(ns *project.Namespaced, targetPath string) (string, error) {
	path, err := r.pathToUse(ns, targetPath)
	if err != nil {
		return "", errs.Wrap(err, "Could not get path to use")
	}

	if fileutils.FileExists(filepath.Join(path, constants.ConfigFileName)) {
		return path, nil
	}

	// If project does not exist at path then we must checkout
	// the project and create the project file
	pj, err := model.FetchProjectByName(ns.Owner, ns.Project)
	if err != nil {
		return "", locale.WrapError(err, "err_fetch_project", "", ns.String())
	}

	branch, err := model.DefaultBranchForProject(pj)
	if err != nil {
		return "", errs.Wrap(err, "Could not grab branch for project")
	}
	branchName := branch.Label

	commitID := ns.CommitID
	if commitID == nil {
		branch, err := model.BranchForProjectByName(pj, branchName)
		if err != nil {
			return "", locale.WrapError(err, "err_fetch_branch", "", branchName)
		}
		commitID = branch.CommitID
	}

	if commitID == nil {
		return "", errs.New("commitID is nil")
	}

	// Clone the related repo, if it is defined
	if pj.RepoURL != nil && *pj.RepoURL != "" {
		err := r.repo.CloneProject(ns.Owner, ns.Project, path, r.Outputer, r.analytics)
		if err != nil {
			return "", locale.WrapError(err, "err_clone_project", "Could not clone associated git repository")
		}
	}

	language, err := getLanguage(*commitID)
	if err != nil {
		return "", errs.Wrap(err, "Could not get language from commitID")
	}

	// Create the config file, if the repo clone didn't already create it
	configFile := filepath.Join(path, constants.ConfigFileName)
	if !fileutils.FileExists(configFile) {
		_, err = projectfile.Create(&projectfile.CreateParams{
			Owner:      ns.Owner,
			Project:    ns.Project,
			CommitID:   commitID,
			BranchName: branchName,
			Directory:  path,
			Language:   language.String(),
		})
		if err != nil {
			return "", errs.Wrap(err, "Could not create projectfile")
		}
	}

	return path, nil
}

func getLanguage(commitID strfmt.UUID) (language.Language, error) {
	modelLanguage, err := model.LanguageByCommit(commitID)
	if err != nil {
		return language.Unset, locale.WrapError(err, "err_language_by_commit", "", string(commitID))
	}

	lang, err := language.MakeByNameAndVersion(modelLanguage.Name, modelLanguage.Version)
	if err != nil {
		return language.Unset, locale.WrapError(err, "err_make_language")
	}
	return lang, nil
}

func (r *Checkout) pathToUse(namespace *project.Namespaced, preferredPath string) (string, error) {
	switch {
	case namespace != nil && namespace.String() != "":
		// Checkout via namespace (eg. state activate org/project) and set resulting path
		return ensureProjectPath(r.config, namespace, preferredPath)
	case preferredPath != "":
		// Use the user provided path
		return preferredPath, nil
	default:
		// Get path from working directory
		targetPath, err := projectfile.GetProjectFilePath()
		return filepath.Dir(targetPath), err
	}
}
