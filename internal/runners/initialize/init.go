package initialize

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/ActiveState/cli/internal/runbits/runtime"
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
	"github.com/ActiveState/cli/internal/runbits/rationalize"
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
	Namespace   string
	ParsedNS    *project.Namespaced
	ProjectName string
	Path        string
	Language    string
	Private     bool
}

// Initialize stores scope-related dependencies.
type Initialize struct {
	auth      *authentication.Auth
	config    Configurable
	out       output.Outputer
	analytics analytics.Dispatcher
	svcModel  *model.SvcModel
}

type Configurable interface {
	projectfile.ConfigGetter
	GetBool(key string) bool
}

type primeable interface {
	primer.Auther
	primer.Configurer
	primer.Outputer
	primer.Analyticer
	primer.SvcModeler
}

type errProjectExists struct {
	error
	path string
}

var errNoOwner = errs.New("Could not find organization")

var errNoLanguage = errs.New("No language specified")

type errUnrecognizedLanguage struct {
	error
	Name string
}

// New returns a prepared ptr to Initialize instance.
func New(prime primeable) *Initialize {
	return &Initialize{prime.Auth(), prime.Config(), prime.Output(), prime.Analytics(), prime.SvcModel()}
}

// inferLanguage tries to infer a reasonable default language from the project currently in use
// (i.e. `state use show`).
// Error handling is not necessary because it's an input error to not include a language to
// `state init`. We're just trying to infer one as a convenience to the user.
func inferLanguage(config projectfile.ConfigGetter, auth *authentication.Auth) (string, string, bool) {
	defaultProjectDir := config.GetString(constants.GlobalDefaultPrefname)
	if defaultProjectDir == "" {
		return "", "", false
	}
	defaultProj, err := project.FromPath(defaultProjectDir)
	if err != nil {
		return "", "", false
	}
	commitID, err := localcommit.Get(defaultProj.Dir())
	if err != nil {
		multilog.Error("Unable to get local commit: %v", errs.JoinMessage(err))
		return "", "", false
	}
	if commitID == "" {
		return "", "", false
	}
	lang, err := model.FetchLanguageForCommit(commitID, auth)
	if err != nil {
		return "", "", false
	}
	return lang.Name, lang.Version, true
}

func (r *Initialize) Run(params *RunParams) (rerr error) {
	logging.Debug("Init: %s %v", params.Namespace, params.Private)

	var (
		paramOwner          string
		paramProjectName    string
		resolvedOwner       string
		resolvedProjectName string
	)
	if params.ParsedNS != nil && params.ParsedNS.IsValid() {
		paramOwner = params.ParsedNS.Owner
		paramProjectName = params.ParsedNS.Project
	} else {
		paramProjectName = params.ProjectName
	}

	defer func() {
		rationalizeError(resolvedOwner, resolvedProjectName, &rerr)
	}()

	if !r.auth.Authenticated() {
		return rationalize.ErrNotAuthenticated
	}

	path := params.Path
	if path == "" {
		var err error
		path, err = osutils.Getwd()
		if err != nil {
			return errs.Wrap(err, "Unable to get current working directory")
		}
	}

	if fileutils.TargetExists(filepath.Join(path, constants.ConfigFileName)) {
		return &errProjectExists{
			error: errs.New("Project file already exists"),
			path:  path,
		}
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
		languageName, languageVersion, inferred = inferLanguage(r.config, r.auth)
	}

	if languageName == "" {
		return errNoLanguage
	}

	// Require 'python', 'python@3', or 'python@2' instead of 'python3' or 'python2'.
	if languageName == language.Python3.String() || languageName == language.Python2.String() {
		return &errUnrecognizedLanguage{Name: languageName}
	}

	lang := language.MakeByNameAndVersion(languageName, languageVersion)
	if !lang.Recognized() {
		return &errUnrecognizedLanguage{Name: languageName}
	}

	version, err := deriveVersion(lang, languageVersion, r.auth)
	if err != nil {
		if inferred || !locale.IsInputError(err) {
			return locale.WrapError(err, "err_init_lang", "", languageName, languageVersion)
		} else {
			return locale.WrapInputError(err, "err_init_lang", "", languageName, languageVersion)
		}
	}

	resolvedOwner, err = r.getOwner(paramOwner)
	if err != nil {
		return errs.Wrap(err, "Unable to determine owner")
	}
	resolvedProjectName = r.getProjectName(paramProjectName, lang.String())
	namespace := project.Namespaced{Owner: resolvedOwner, Project: resolvedProjectName}

	r.out.Notice(locale.T("initializing_project"))

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

	bp := model.NewBuildPlannerModel(r.auth)
	commitID, err := bp.CreateProject(&model.CreateProjectParams{
		Owner:       namespace.Owner,
		Project:     namespace.Project,
		PlatformID:  strfmt.UUID(platformID),
		Language:    lang.Requirement(),
		Version:     version,
		Private:     params.Private,
		Description: locale.T("commit_message_add_initial"),
	})
	if err != nil {
		return locale.WrapError(err, "err_init_commit", "Could not create project")
	}

	if err := localcommit.Set(proj.Dir(), commitID.String()); err != nil {
		return errs.Wrap(err, "Unable to create local commit file")
	}

	_, err = runtime.SolveAndUpdate(r.auth, r.out, r.analytics, proj, &commitID, target.TriggerInit, r.svcModel, r.config, runtime.OptOrderChanged)
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

	initSuccessMsg := locale.Tr("init_success", namespace.String(), path, executables)
	if !strings.EqualFold(paramOwner, resolvedOwner) {
		initSuccessMsg = locale.Tr("init_success_resolved_owner", namespace.String(), path, executables)
	}

	r.out.Print(output.Prepare(
		initSuccessMsg,
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

func getKnownVersions(lang language.Language, auth *authentication.Auth) ([]string, error) {
	pkgs, err := model.SearchIngredientsStrict(model.NewNamespaceLanguage().String(), lang.Requirement(), false, true, nil, auth)
	if err != nil {
		return nil, errs.Wrap(err, "Failed to fetch Platform languages")
	}

	if len(pkgs) == 0 {
		return nil, &errUnrecognizedLanguage{Name: lang.Requirement()}
	}

	knownVersions := make([]string, len(pkgs))
	for i, pkg := range pkgs {
		knownVersions[i] = pkg.Version
	}
	return knownVersions, nil
}

var versionRe = regexp.MustCompile(`^\d(\.\d+)*$`)

func deriveVersion(lang language.Language, version string, auth *authentication.Auth) (string, error) {
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

	// If a bare version number was given, and if it is a partial version number (e.g. python@3.10),
	// append a '.x' suffix.
	if versionRe.MatchString(version) {
		knownVersions, err := getKnownVersions(lang, auth)
		if err != nil {
			return "", errs.Wrap(err, "Unable to get known versions for language %s", lang.Requirement())
		}

		validVersionPrefix := false
		for _, knownVersion := range knownVersions {
			if knownVersion == version {
				return version, nil // e.g. python@3.10.10
			} else if strings.HasPrefix(knownVersion, version) {
				validVersionPrefix = true // not an exact match, e.g. python@3.10
			}
		}

		if validVersionPrefix {
			version += ".x"
		}
	}

	return version, nil
}

func (i *Initialize) getOwner(desiredOwner string) (string, error) {
	orgs, err := model.FetchOrganizations(i.auth)
	if err != nil {
		return "", errs.Wrap(err, "Unable to get the user's writable orgs")
	}

	// Prefer the desired owner if it's valid
	if desiredOwner != "" {
		// Match the case of the organization.
		// Otherwise the incorrect case will be written to the project file.
		for _, org := range orgs {
			if strings.EqualFold(org.URLname, desiredOwner) {
				return org.URLname, nil
			}
		}
		// Return desiredOwner for error reporting
		return desiredOwner, errNoOwner
	}

	// Use the last used namespace if it's valid
	lastUsed := i.config.GetString(constants.LastUsedNamespacePrefname)
	if lastUsed != "" {
		ns, err := project.ParseNamespace(lastUsed)
		if err != nil {
			return "", errs.Wrap(err, "Unable to parse last used namespace")
		}

		for _, org := range orgs {
			if strings.EqualFold(org.URLname, ns.Owner) {
				return org.URLname, nil
			}
		}
	}

	// Use the first org if there is one
	if len(orgs) > 0 {
		return orgs[0].URLname, nil
	}

	return "", errNoOwner
}

func (i *Initialize) getProjectName(desiredProject string, lang string) string {
	if desiredProject != "" {
		return desiredProject
	}

	return fmt.Sprintf("%s-%s", lang, model.HostPlatform)
}
