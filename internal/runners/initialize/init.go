package initialize

import (
	"os"
	"path/filepath"
	"strings"

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
	commitID := defaultProj.CommitUUID()
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

	createParams := &projectfile.CreateParams{
		Owner:     params.Namespace.Owner,
		Project:   params.Namespace.Project,
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

	commitID, err := model.CommitInitial(model.HostPlatform, lang.Requirement(), version)
	if err != nil {
		return locale.WrapError(err, "err_init_commit", "Could not create initial commit")
	}

	if err := proj.SetCommit(commitID.String()); err != nil {
		return locale.WrapError(err, "err_init_setcommit", "Could not store commit to project file")
	}

	logging.Debug("Creating Platform project and pushing it")

	platformProject, err := model.CreateEmptyProject(params.Namespace.Owner, params.Namespace.Project, params.Private)
	if err != nil {
		return locale.WrapInputError(err, "err_init_create_project", "Failed to create a Platform project at {{.V0}}.", params.Namespace.String())
	}

	branch, err := model.DefaultBranchForProject(platformProject) // only one branch for newly created project
	if err != nil {
		return locale.NewInputError("err_no_default_branch")
	}

	err = model.UpdateProjectBranchCommitWithModel(platformProject, branch.Label, commitID)
	if err != nil {
		return locale.WrapError(err, "err_init_push", "Failed to push to the newly created Platform project at {{.V0}}", params.Namespace.String())
	}

	err = runbits.RefreshRuntime(r.auth, r.out, r.analytics, proj, commitID, true, target.TriggerInit, r.svcModel)
	if err != nil {
		return locale.WrapError(err, "err_init_refresh", "Could not setup runtime after init")
	}

	projectfile.StoreProjectMapping(r.config, params.Namespace.String(), filepath.Dir(proj.Source().Path()))

	projectTarget := target.NewProjectTarget(proj, nil, "").Dir()
	executables := setup.ExecDir(projectTarget)

	r.out.Print(output.Prepare(
		locale.Tr("init_success", params.Namespace.String(), path, executables),
		&struct {
			Namespace   string `json:"namespace"`
			Path        string `json:"path" `
			Executables string `json:"executables"`
		}{
			params.Namespace.String(),
			path,
			executables,
		},
	))

	return nil
}

func getKnownVersionsFromPlatform(lang language.Language) ([]string, error) {
	pkgs, err := model.SearchIngredientsStrict(model.NewNamespaceLanguage(), lang.Requirement(), false, true)
	if err != nil {
		return nil, locale.WrapError(err, "err_init_verify_language", "Inventory search failed unexpectedly")
	}

	if len(pkgs) == 0 {
		return nil, locale.NewInputError("err_init_language_not_found", "The selected language cannot be found")
	}

	knownVersions := make([]string, len(pkgs))
	for i, pkg := range pkgs {
		knownVersions[i] = pkg.Version
	}
	return knownVersions, nil
}

// Can be overridden in tests.
var getKnownVersions func(language.Language) ([]string, error) = getKnownVersionsFromPlatform

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

	// Fetch known list of languages and verify the given version matches it, either exactly or partially.
	knownVersions, err := getKnownVersions(lang)
	if err != nil {
		return "", errs.Wrap(err, "Unable to get known versions for language %s", lang.Requirement())
	}

	prefix, _, _ := strings.Cut(version, ",") // only keep first part of e.g. ">=3.10,<3.11"
	prefix = strings.TrimLeft(prefix, ">=<")  // strip leading constraint characters
	prefix = strings.TrimSuffix(prefix, ".x") // strip trailing wildcard
	prefix = strings.TrimSuffix(prefix, ".X") // string trailing wildcard
	validVersionPrefix := false
	for _, knownVersion := range knownVersions {
		if knownVersion == version {
			return knownVersion, nil // e.g. python@3.10.10
		} else if strings.HasPrefix(knownVersion, prefix) {
			validVersionPrefix = true // e.g. python@3.10
			break
		}
	}

	if !validVersionPrefix {
		return "", errs.AddTips(
			locale.NewInputError(
				"err_init_language_version_not_found",
				"The selected version of the language cannot be found",
			),
			locale.Tl(
				"version_not_found_check_format",
				"Please ensure that the version format is valid.",
			),
		)
	}

	if prefix == version {
		return version + ".x", nil // not an exact match, e.g. python@3.10
	}

	return version, nil
}
