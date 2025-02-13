package checkout

import (
	"path/filepath"

	"github.com/go-openapi/strfmt"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/language"
	"github.com/ActiveState/cli/internal/osutils"
	"github.com/ActiveState/cli/internal/primer"
	"github.com/ActiveState/cli/internal/runbits/buildscript"
	"github.com/ActiveState/cli/internal/runbits/git"
	"github.com/ActiveState/cli/pkg/localcommit"
	"github.com/ActiveState/cli/pkg/platform/api/mono/mono_models"
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
var ErrNoOrg = errs.New("unable to get org name")

func New(repo git.Repository, prime primeable) *Checkout {
	return &Checkout{repo, prime}
}

func (r *Checkout) Run(ns *project.Namespaced, branchName, cachePath, targetPath string, noClone, bareCheckout, portable bool) (_ string, rerr error) {
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
				return "", errs.Wrap(err, "Could not clone associated git repository")
			}
		}
	} else if commitID == nil {
		return "", errNoCommitID
	}

	if err := CreateProjectFiles(path, cachePath, owner, proj, branchName, commitID.String(), language, portable); err != nil {
		return "", errs.Wrap(err, "Could not create project files")
	}

	if r.prime.Config().GetBool(constants.OptinBuildscriptsConfig) {
		pjf, err := projectfile.FromPath(path)
		if err != nil {
			return "", errs.Wrap(err, "Unable to load project file")
		}

		if err := buildscript_runbit.Initialize(pjf, r.prime.Auth(), r.prime.SvcModel()); err != nil {
			return "", errs.Wrap(err, "Unable to initialize buildscript")
		}
	}

	return path, nil
}

func (r *Checkout) fetchProject(
	ns *project.Namespaced, branchName string, commitID *strfmt.UUID) (string, string, *strfmt.UUID, string, string, *string, error) {

	// If project does not exist at path then we must checkout
	// the project and create the project file
	pj, err := model.FetchProjectByName(ns.Owner, ns.Project, r.prime.Auth())
	if err != nil {
		return "", "", nil, "", "", nil, errs.Wrap(err, "Unable to fetch project '%s'", ns.String())
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
			return "", "", nil, "", "", nil, errs.Wrap(err, "Could not get branch '%s'", branchName)
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
		return "", "", nil, "", "", nil, ErrNoOrg
	}
	owner := owners[0].URLName

	return owner, proj, commitID, branchName, language, pj.RepoURL, nil
}

func CreateProjectFiles(checkoutPath, cachePath, owner, name, branch, commitID, language string, portable bool) error {
	configFile := filepath.Join(checkoutPath, constants.ConfigFileName)
	if !fileutils.FileExists(configFile) {
		_, err := projectfile.Create(&projectfile.CreateParams{
			Owner:      owner,
			Project:    name, // match case on the Platform
			BranchName: branch,
			Directory:  checkoutPath,
			Language:   language,
			Cache:      cachePath,
			Portable:   portable,
		})
		if err != nil {
			if osutils.IsAccessDeniedError(err) {
				return &ErrNoPermission{checkoutPath}
			}
			return errs.Wrap(err, "Could not create projectfile")
		}
	}

	if err := localcommit.Set(checkoutPath, commitID); err != nil {
		return errs.Wrap(err, "Could not create local commit file")
	}

	return nil
}

func getLanguage(commitID strfmt.UUID, auth *authentication.Auth) (language.Language, error) {
	modelLanguage, err := model.LanguageByCommit(commitID, auth)
	if err != nil {
		return language.Unset, errs.Wrap(err, "Could not get language from commit ID '%s'", string(commitID))
	}

	return language.MakeByNameAndVersion(modelLanguage.Name, modelLanguage.Version), nil
}
