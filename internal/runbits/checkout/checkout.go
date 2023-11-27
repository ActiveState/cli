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
	"github.com/ActiveState/cli/internal/multilog"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/runbits/git"
	"github.com/ActiveState/cli/pkg/localcommit"
	"github.com/ActiveState/cli/pkg/platform/authentication"
	"github.com/ActiveState/cli/pkg/platform/model"
	"github.com/ActiveState/cli/pkg/project"
	"github.com/ActiveState/cli/pkg/projectfile"
)

type primeable interface {
	primer.Outputer
	primer.Analyticer
	primer.Configurer
	primer.Auther
}

// Checkout will checkout the given platform project at the given path
// This includes cloning an associated repository and creating the activestate.yaml
// It does not activate any environment
type Checkout struct {
	repo git.Repository
	output.Outputer
	config     *config.Instance
	analytics  analytics.Dispatcher
	branchName string
	auth       *authentication.Auth
}

func New(repo git.Repository, prime primeable) *Checkout {
	return &Checkout{repo, prime.Output(), prime.Config(), prime.Analytics(), "", prime.Auth()}
}

func (r *Checkout) Run(ns *project.Namespaced, branchName, cachePath, targetPath string, noClone bool) (_ string, rerr error) {
	defer r.rationalizeError(&rerr)

	path, err := r.pathToUse(ns, targetPath)
	if err != nil {
		return "", errs.Wrap(err, "Could not get path to use")
	}

	path, err = filepath.Abs(path)
	if err != nil {
		return "", errs.Wrap(err, "Could not get absolute path")
	}

	emptyDir, err := fileutils.IsEmptyDir(path)
	if err != nil {
		multilog.Error("Unable to check if directory is empty: %v", err)
	}

	// If project does not exist at path then we must checkout
	// the project and create the project file
	pj, err := model.FetchProjectByName(ns.Owner, ns.Project)
	if err != nil {
		return "", locale.WrapError(err, "err_fetch_project", "", ns.String())
	}

	if branchName == "" {
		branch, err := model.DefaultBranchForProject(pj)
		if err != nil {
			return "", errs.Wrap(err, "Could not grab branch for project")
		}
		branchName = branch.Label
	}

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
	if !noClone && pj.RepoURL != nil && *pj.RepoURL != "" {
		err := r.repo.CloneProject(ns.Owner, ns.Project, path, r.Outputer, r.analytics)
		if err != nil {
			return "", locale.WrapError(err, "err_clone_project", "Could not clone associated git repository")
		}
	}

	language, err := getLanguage(*commitID)
	if err != nil {
		return "", errs.Wrap(err, "Could not get language from commitID")
	}

	if cachePath != "" && !filepath.IsAbs(cachePath) {
		cachePath, err = filepath.Abs(cachePath)
		if err != nil {
			return "", errs.Wrap(err, "Could not get absolute path for cache")
		}
	}

	// Match the case of the organization.
	// Otherwise the incorrect case will be written to the project file.
	owners, err := model.FetchOrganizationsByIDs([]strfmt.UUID{pj.OrganizationID})
	if err != nil {
		return "", errs.Wrap(err, "Unable to get the project's org")
	}
	if len(owners) == 0 {
		return "", locale.NewInputError("err_no_org_name", "Your project's organization name could not be found")
	}
	owner := owners[0].URLName

	// Create the config file, if the repo clone didn't already create it
	configFile := filepath.Join(path, constants.ConfigFileName)
	if !fileutils.FileExists(configFile) {
		_, err = projectfile.Create(&projectfile.CreateParams{
			Owner:      owner,
			Project:    pj.Name, // match case on the Platform
			BranchName: branchName,
			Directory:  path,
			Language:   language.String(),
			Cache:      cachePath,
		})
		if err != nil {
			return "", errs.Wrap(err, "Could not create projectfile")
		}
	}

	err = localcommit.Set(path, commitID.String())
	if err != nil {
		return "", errs.Wrap(err, "Could not create local commit file")
	}
	if emptyDir || fileutils.DirExists(filepath.Join(path, ".git")) {
		err = localcommit.AddToGitIgnore(path)
		if err != nil {
			r.Outputer.Notice(locale.Tr("notice_commit_id_gitignore", constants.ProjectConfigDirName, constants.CommitIdFileName))
			multilog.Error("Unable to add local commit file to .gitignore: %v", err)
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
