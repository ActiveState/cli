package checkout

import (
	"path/filepath"

	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/primer"
	"github.com/go-openapi/strfmt"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/language"
	"github.com/ActiveState/cli/internal/osutils"
	"github.com/ActiveState/cli/internal/runbits/git"
	"github.com/ActiveState/cli/pkg/platform/api/mono/mono_models"
	"github.com/ActiveState/cli/pkg/platform/authentication"
	"github.com/ActiveState/cli/pkg/platform/model"
	"github.com/ActiveState/cli/pkg/platform/model/buildplanner"
	"github.com/ActiveState/cli/pkg/project"
	"github.com/ActiveState/cli/pkg/projectfile"
)

type primeable interface {
	primer.Outputer
	primer.Analyticer
	primer.Configurer
	primer.Auther
	primer.SvcModeler
}

// Checkout will checkout the given platform project at the given path
// This includes cloning an associated repository and creating the activestate.yaml
// It does not activate any environment
type Checkout struct {
	repo  git.Repository
	prime primeable
}

type errCommitDoesNotBelong struct {
	CommitID strfmt.UUID
}

func (e errCommitDoesNotBelong) Error() string {
	return "commitID does not belong to the given branch"
}

var errNoCommitID = errs.New("commitID is nil")

func New(repo git.Repository, prime primeable) *Checkout {
	return &Checkout{repo, prime}
}

func (r *Checkout) Run(ns *project.Namespaced, branchName, cachePath, targetPath string, noClone, bareCheckout bool) (_ string, rerr error) {
	defer r.rationalizeError(&rerr)

	path, err := r.pathToUse(ns, targetPath)
	if err != nil {
		return "", errs.Wrap(err, "Could not get path to use")
	}

	path, err = filepath.Abs(path)
	if err != nil {
		return "", errs.Wrap(err, "Could not get absolute path")
	}

	if cachePath != "" && !filepath.IsAbs(cachePath) {
		cachePath, err = filepath.Abs(cachePath)
		if err != nil {
			return "", errs.Wrap(err, "Could not get absolute path for cache")
		}
	}

	owner := ns.Owner
	proj := ns.Project
	commitID := ns.CommitID
	var language string
	if !bareCheckout {
		var repoURL *string
		owner, proj, commitID, branchName, language, repoURL, err = r.fetchProject(ns, branchName, commitID)
		if err != nil {
			return "", errs.Wrap(err, "Unable to checkout project")
		}

		// Clone the related repo, if it is defined
		if !noClone && repoURL != nil && *repoURL != "" {
			err := r.repo.CloneProject(ns.Owner, ns.Project, path, r.prime.Output(), r.prime.Analytics())
			if err != nil {
				return "", locale.WrapError(err, "err_clone_project", "Could not clone associated git repository")
			}
		}
	} else if commitID == nil {
		return "", errNoCommitID
	}

	if err := CreateProjectFiles(path, cachePath, owner, proj, branchName, commitID.String(), language); err != nil {
		return "", errs.Wrap(err, "Could not create project files")
	}

	bp := buildplanner.NewBuildPlannerModel(r.prime.Auth(), r.prime.SvcModel())
	script, err := bp.GetBuildScript(owner, proj, branchName, commitID.String())
	if err != nil {
		return "", errs.Wrap(err, "Unable to get the remote build script")
	}
	err = script.Write(path)
	if err != nil {
		return "", errs.Wrap(err, "Failed to write buildscript")
	}

	return path, nil
}

func (r *Checkout) fetchProject(
	ns *project.Namespaced, branchName string, commitID *strfmt.UUID) (string, string, *strfmt.UUID, string, string, *string, error) {

	// If project does not exist at path then we must checkout
	// the project and create the project file
	pj, err := model.FetchProjectByName(ns.Owner, ns.Project, r.prime.Auth())
	if err != nil {
		return "", "", nil, "", "", nil, locale.WrapError(err, "err_fetch_project", "", ns.String())
	}
	proj := pj.Name

	var branch *mono_models.Branch

	switch {
	// Fetch the branch the given commitID is on.
	case commitID != nil:
		for _, b := range pj.Branches {
			if belongs, err := model.CommitBelongsToBranch(ns.Owner, ns.Project, b.Label, *commitID, r.prime.Auth()); err == nil && belongs {
				branch = b
				break
			} else if err != nil {
				return "", "", nil, "", "", nil, errs.Wrap(err, "Could not determine which branch the given commitID belongs to")
			}
		}
		if branch == nil {
			return "", "", nil, "", "", nil, &errCommitDoesNotBelong{CommitID: *commitID}
		}

	// Fetch the given project branch.
	case branchName != "":
		branch, err = model.BranchForProjectByName(pj, branchName)
		if err != nil {
			return "", "", nil, "", "", nil, locale.WrapError(err, "err_fetch_branch", "", branchName)
		}
		commitID = branch.CommitID

	// Fetch the default branch for the given project.
	default:
		branch, err = model.DefaultBranchForProject(pj)
		if err != nil {
			return "", "", nil, "", "", nil, errs.Wrap(err, "Could not grab branch for project")
		}
		commitID = branch.CommitID
	}
	branchName = branch.Label

	if commitID == nil {
		return "", "", nil, "", "", nil, errNoCommitID
	}

	lang, err := getLanguage(*commitID, r.prime.Auth())
	if err != nil {
		return "", "", nil, "", "", nil, errs.Wrap(err, "Could not get language from commitID")
	}
	language := lang.String()

	// Match the case of the organization.
	// Otherwise the incorrect case will be written to the project file.
	owners, err := model.FetchOrganizationsByIDs([]strfmt.UUID{pj.OrganizationID}, r.prime.Auth())
	if err != nil {
		return "", "", nil, "", "", nil, errs.Wrap(err, "Unable to get the project's org")
	}
	if len(owners) == 0 {
		return "", "", nil, "", "", nil, locale.NewInputError("err_no_org_name", "Your project's organization name could not be found")
	}
	owner := owners[0].URLName

	return owner, proj, commitID, branchName, language, pj.RepoURL, nil
}

func CreateProjectFiles(checkoutPath, cachePath, owner, name, branch, commitID, language string) error {
	configFile := filepath.Join(checkoutPath, constants.ConfigFileName)
	if !fileutils.FileExists(configFile) {
		_, err := projectfile.Create(&projectfile.CreateParams{
			Owner:      owner,
			Project:    name, // match case on the Platform
			BranchName: branch,
			CommitID:   commitID,
			Directory:  checkoutPath,
			Language:   language,
			Cache:      cachePath,
		})
		if err != nil {
			if osutils.IsAccessDeniedError(err) {
				return &ErrNoPermission{checkoutPath}
			}
			return errs.Wrap(err, "Could not create projectfile")
		}
	}

	return nil
}

func getLanguage(commitID strfmt.UUID, auth *authentication.Auth) (language.Language, error) {
	modelLanguage, err := model.LanguageByCommit(commitID, auth)
	if err != nil {
		return language.Unset, locale.WrapError(err, "err_language_by_commit", "", string(commitID))
	}

	return language.MakeByNameAndVersion(modelLanguage.Name, modelLanguage.Version), nil
}
