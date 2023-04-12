package initialize

import (
	"path/filepath"
	"strings"

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
	"github.com/ActiveState/cli/pkg/platform/model"
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
	config projectfile.ConfigGetter
	out    output.Outputer
}

type primeable interface {
	primer.Configurer
	primer.Outputer
}

// New returns a prepared ptr to Initialize instance.
func New(prime primeable) *Initialize {
	return &Initialize{prime.Config(), prime.Output()}
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

func (r *Initialize) Run(params *RunParams) error {
	if err := params.Namespace.Validate(); err != nil {
		return locale.WrapInputError(err, "init_invalid_namespace_err", "The provided namespace argument is invalid.")
	}

	logging.Debug("Init: %s/%s %v", params.Namespace.Owner, params.Namespace.Project, params.Private)

	path := params.Path
	if path == "" {
		var err error
		path, err = osutils.Getwd()
		if err != nil {
			return locale.WrapInputError(err, "err_init_sanitize_path", "Could not prepare path: {{.V0}}", err.Error())
		}
	}

	if fileutils.FileExists(filepath.Join(path, constants.ConfigFileName)) {
		return locale.NewInputError("err_projectfile_exists")
	}

	if _, err := fileutils.PrepareDir(path); err != nil {
		return locale.WrapError(err, "err_init_preparedir", "Could not create directory at [NOTICE]{{.V0}}[/RESET]. Error: {{.V1}}", params.Path, err.Error())
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
		return locale.NewInputError("err_init_no_language", "You need to supply the [NOTICE]language[/RESET] argument, run [ACTIONABLE]`state init --help`[/RESET] for more information.")
	}

	lang, err := language.MakeByNameAndVersion(languageName, languageVersion)
	if err != nil {
		if inferred {
			return locale.WrapError(err, "err_init_lang", "", languageName, languageVersion)
		} else {
			return locale.WrapInputError(err, "err_init_lang", "", languageName, languageVersion)
		}
	}
	err = lang.Validate()
	if err != nil {
		if inferred {
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

	proj, err := project.New(pjfile, r.out)
	if err != nil {
		return err
	}

	version := deriveVersion(lang, languageVersion)
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

	projectfile.StoreProjectMapping(r.config, params.Namespace.String(), filepath.Dir(proj.Source().Path()))

	r.out.Notice(locale.Tr(
		"init_success",
		params.Namespace.Owner,
		params.Namespace.Project,
		path,
	))

	return nil
}

func deriveVersion(lang language.Language, version string) string {
	if version != "" {
		return version
	}

	langs, err := model.FetchSupportedLanguages(model.HostPlatform)
	if err != nil {
		multilog.Error("Failed to fetch supported languages (using hardcoded default version): %s", errs.JoinMessage(err))
		return lang.RecommendedVersion()
	}

	for _, l := range langs {
		if lang.String() == l.Name || (lang == language.Python3 && l.Name == "python") {
			return l.DefaultVersion
		}
	}

	multilog.Error("Could not find requested language in fetched languages (using hardcoded default version): %s", lang)
	return lang.RecommendedVersion()
}
