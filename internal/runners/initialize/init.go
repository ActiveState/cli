package initialize

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/go-openapi/strfmt"

	"github.com/ActiveState/cli/internal/analytics"
	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/language"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/multilog"
	"github.com/ActiveState/cli/internal/osutils"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/primer"
	"github.com/ActiveState/cli/internal/runbits"
	"github.com/ActiveState/cli/internal/runbits/commitmediator"
	"github.com/ActiveState/cli/pkg/localcommit"
	"github.com/ActiveState/cli/pkg/platform/authentication"
	"github.com/ActiveState/cli/pkg/platform/model"
	"github.com/ActiveState/cli/pkg/platform/runtime/setup"
	"github.com/ActiveState/cli/pkg/platform/runtime/target"
	"github.com/ActiveState/cli/pkg/project"
	"github.com/ActiveState/cli/pkg/projectfile"
)

// RunParams stores run func parameters.
type RunParams struct {
	Namespace *project.Namespaced
	Path      string
	Language  string
	Private   bool
}

// Initialize stores scope-related dependencies.
type Initialize struct {
	auth      *authentication.Auth
	config    projectfile.ConfigGetter
	out       output.Outputer
	analytics analytics.Dispatcher
	svcModel  *model.SvcModel
}

type primeable interface {
	primer.Auther
	primer.Configurer
	primer.Outputer
	primer.Analyticer
	primer.SvcModeler
}

// New returns a prepared ptr to Initialize instance.
func New(prime primeable) *Initialize {
	return &Initialize{prime.Auth(), prime.Config(), prime.Output(), prime.Analytics(), prime.SvcModel()}
}

// inferLanguage tries to infer a reasonable default language from the project currently in use
// (i.e. `state use show`).
// Error handling is not necessary because it's an input error to not include a language to
// `state init`. We're just trying to infer one as a convenience to the user.
func inferLanguage(config projectfile.ConfigGetter) (string, string, bool) {
	defaultProjectDir := config.GetString(constants.GlobalDefaultPrefname)
	if defaultProjectDir == "" {
		return "", "", false
	}
	defaultProj, err := project.FromPath(defaultProjectDir)
	if err != nil {
		return "", "", false
	}
	commitID, err := commitmediator.Get(defaultProj)
	if err != nil {
		multilog.Error("Unable to get local commit: %v", errs.JoinMessage(err))
		return "", "", false
	}
	if commitID == "" {
		return "", "", false
	}
	lang, err := model.FetchLanguageForCommit(commitID)
	if err != nil {
		return "", "", false
	}
	return lang.Name, lang.Version, true
}

func (r *Initialize) Run(params *RunParams) (rerr error) {
	defer rationalizeError(&rerr)
	logging.Debug("Init: %s/%s %v", params.Namespace.Owner, params.Namespace.Project, params.Private)

	if !r.auth.Authenticated() {
		return locale.NewInputError("err_init_authenticated")
	}

	path := params.Path
	if path == "" {
		var err error
		path, err = osutils.Getwd()
		if err != nil {
			return locale.WrapInputError(err, "err_init_sanitize_path", "Could not prepare path: {{.V0}}", err.Error())
		}
	}

	if fileutils.TargetExists(filepath.Join(path, constants.ConfigFileName)) {
		return locale.NewInputError("err_projectfile_exists")
	}

	err := fileutils.MkdirUnlessExists(path)
	if err != nil {
		return locale.WrapError(err, "err_init_preparedir", "Could not create directory at [NOTICE]{{.V0}}[/RESET]. Error: {{.V1}}", params.Path, err.Error())
	}

	path, err = filepath.Abs(params.Path)
	if err != nil {
		return locale.WrapInputError(err, "err_init_abs_path", "Could not determine absolute path to [NOTICE]{{.V0}}[/RESET]. Error: {{.V1}}", path, err.Error())
	}

	var languageName, languageVersion string
	var inferred bool
	if params.Language != "" {
		langParts := strings.Split(params.Language, "@")
		languageName = langParts[0]
		if len(langParts) > 1 {
			languageVersion = langParts[1]
		}
	} else {
		languageName, languageVersion, inferred = inferLanguage(r.config)
	}

	if languageName == "" {
		return locale.NewInputError("err_init_no_language")
	}

	// Require 'python', 'python@3', or 'python@2' instead of 'python3' or 'python2'.
	if languageName == language.Python3.String() || languageName == language.Python2.String() {
		return language.UnrecognizedLanguageError(languageName, language.RecognizedSupportedsNames())
	}

	lang, err := language.MakeByNameAndVersion(languageName, languageVersion)
	if err != nil {
		if inferred {
			return locale.WrapError(err, "err_init_lang", "", languageName, languageVersion)
		} else {
			return locale.WrapInputError(err, "err_init_lang", "", languageName, languageVersion)
		}
	}

	version, err := deriveVersion(lang, languageVersion)
	if err != nil {
		if inferred || !locale.IsInputError(err) {
			return locale.WrapError(err, "err_init_lang", "", languageName, languageVersion)
		} else {
			return locale.WrapInputError(err, "err_init_lang", "", languageName, languageVersion)
		}
	}

	emptyDir, err := fileutils.IsEmptyDir(path)
	if err != nil {
		multilog.Error("Unable to check if directory is empty: %v", err)
	}

	// Match the case of the organization.
	// Otherwise the incorrect case will be written to the project file.
	var owner string
	orgs, err := model.FetchOrganizations()
	if err != nil {
		return errs.Wrap(err, "Unable to get the user's writable orgs")
	}
	for _, org := range orgs {
		if strings.EqualFold(org.URLname, params.Namespace.Owner) {
			owner = org.URLname
			break
		}
	}
	if owner == "" {
		return locale.NewInputError("err_invalid_org",
			"The organization '[ACTIONABLE]{{.V0}}[/RESET]' either does not exist, or you do not have permissions to create a project in it.",
			params.Namespace.Owner)
	}
	namespace := project.Namespaced{Owner: owner, Project: params.Namespace.Project}

	createParams := &projectfile.CreateParams{
		Owner:     namespace.Owner,
		Project:   namespace.Project,
		Language:  lang.String(),
		Directory: path,
		Private:   params.Private,
	}

	pjfile, err := projectfile.Create(createParams)
	if err != nil {
		return locale.WrapError(err, "err_init_pjfile", "Could not create project file")
	}

	// If an error occurs, remove the created activestate.yaml file so the user can try again.
	defer func() {
		if rerr == nil {
			return
		}
		err := os.Remove(pjfile.Path())
		if err != nil {
			multilog.Error("Failed to remove activestate.yaml after `state init` error: %v", err)
			return
		}
		if cwd, err := osutils.Getwd(); err == nil {
			if createdDir := filepath.Dir(pjfile.Path()); createdDir != cwd {
				err2 := os.RemoveAll(createdDir)
				if err2 != nil {
					multilog.Error("Failed to remove created directory after `state init` error: %v", err2)
				}
			}
		}
	}()

	proj, err := project.New(pjfile, r.out)
	if err != nil {
		return err
	}

	logging.Debug("Creating Platform project")

	platformID, err := model.PlatformNameToPlatformID(model.HostPlatform)
	if err != nil {
		return errs.Wrap(err, "Unable to determine Platform ID from %s", model.HostPlatform)
	}

	timestamp, err := model.FetchLatestTimeStamp()
	if err != nil {
		return errs.Wrap(err, "Unable to fetch latest timestamp")
	}

	bp := model.NewBuildPlannerModel(r.auth)
	commitID, err := bp.CreateProject(&model.CreateProjectParams{
		Owner:       namespace.Owner,
		Project:     namespace.Project,
		PlatformID:  strfmt.UUID(platformID),
		Language:    lang.Requirement(),
		Version:     version,
		Private:     params.Private,
		Timestamp:   *timestamp,
		Description: locale.T("commit_message_add_initial"),
	})
	if err != nil {
		return locale.WrapError(err, "err_init_commit", "Could not create initial commit")
	}

	if err := commitmediator.Set(proj, commitID.String()); err != nil {
		return errs.Wrap(err, "Unable to create local commit file")
	}
	if emptyDir || fileutils.DirExists(filepath.Join(path, ".git")) {
		err := localcommit.AddToGitIgnore(path)
		if err != nil {
			r.out.Notice(locale.Tr("notice_commit_id_gitignore", constants.ProjectConfigDirName, constants.CommitIdFileName))
			multilog.Error("Unable to add local commit file to .gitignore: %v", err)
		}
	}

	err = runbits.RefreshRuntime(r.auth, r.out, r.analytics, proj, commitID, true, target.TriggerInit, r.svcModel)
	if err != nil {
		logging.Debug("Deleting remotely created project due to runtime setup error")
		err2 := model.DeleteProject(namespace.Owner, namespace.Project, r.auth)
		if err2 != nil {
			multilog.Error("Error deleting remotely created project after runtime setup error: %v", errs.JoinMessage(err2))
			return locale.WrapError(err, "err_init_refresh_delete_project", "Could not setup runtime after init, and could not delete newly created Platform project. Please delete it manually before trying again")
		}
		return locale.WrapError(err, "err_init_refresh", "Could not setup runtime after init")
	}

	projectfile.StoreProjectMapping(r.config, namespace.String(), filepath.Dir(proj.Source().Path()))

	projectTarget := target.NewProjectTarget(proj, nil, "").Dir()
	executables := setup.ExecDir(projectTarget)

	r.out.Print(output.Prepare(
		locale.Tr("init_success", namespace.String(), path, executables),
		&struct {
			Namespace   string `json:"namespace"`
			Path        string `json:"path" `
			Executables string `json:"executables"`
		}{
			namespace.String(),
			path,
			executables,
		},
	))

	return nil
}

func deriveVersion(lang language.Language, version string) (string, error) {
	err := lang.Validate()
	if err != nil {
		return "", errs.Wrap(err, "Failed to validate language")
	}

	if version == "" {
		// Return default language.
		langs, err := model.FetchSupportedLanguages(model.HostPlatform)
		if err != nil {
			multilog.Error("Failed to fetch supported languages (using hardcoded default version): %s", errs.JoinMessage(err))
			return lang.RecommendedVersion(), nil
		}

		for _, l := range langs {
			if lang.String() == l.Name || (lang == language.Python3 && l.Name == language.Python3.Requirement()) {
				return l.DefaultVersion, nil
			}
		}

		multilog.Error("Could not find requested language in fetched languages (using hardcoded default version): %s", lang)
		return lang.RecommendedVersion(), nil
	}

	return version, nil
}
