package initialize

import (
	"fmt"
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
	language  language.Supported
	version   string
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

func sanitize(params *RunParams, config projectfile.ConfigGetter) error {
	// Try to infer the language from the project currently in use (i.e. `state use show`).
	// Error handling is not necessary because it's an input error to not include a language to
	// `state init`. We're just trying to infer one as a convenience to the user.
	if params.Language == "" {
		if projectDir := config.GetString(constants.GlobalDefaultPrefname); projectDir != "" {
			if proj, err := project.FromPath(projectDir); err == nil {
				if commitID := proj.CommitUUID(); commitID != "" {
					if lang, err := model.FetchLanguageForCommit(commitID); err == nil {
						params.Language = fmt.Sprintf("%s@%s", lang.Name, lang.Version)
					}
				}
			}
		}
	}

	if params.Language == "" {
		// Manually check for language requirement, because we need to fallback on the --language flag to support editor.V0
		return locale.NewInputError("err_init_no_language", "You need to supply the [NOTICE]language[/RESET] argument, run [ACTIONABLE]`state init --help`[/RESET] for more information.")
	}
	langParts := strings.Split(params.Language, "@")
	if len(langParts) > 1 {
		params.version = langParts[1]
	}

	// Disambiguate "python", defaulting to "python3" if no version was given.
	if langParts[0] == "python" {
		if params.version == "" || strings.HasPrefix(params.version, "3") {
			langParts[0] = "python3"
		} else if strings.HasPrefix(params.version, "2") {
			langParts[0] = "python2"
		}
	}

	params.language = language.Supported{language.MakeByName(langParts[0])}
	if !params.language.Recognized() {
		return language.UnrecognizedLanguageError(
			params.Language,
			language.RecognizedSupportedsNames(),
		)
	}

	// Fail if target dir already has an activestate.yaml
	if fileutils.FileExists(filepath.Join(params.Path, constants.ConfigFileName)) {
		absPath, err := filepath.Abs(params.Path)
		if err != nil {
			return errs.Wrap(err, "IO failure")
		}
		return locale.NewInputError("err_init_file_exists", "", absPath)
	}

	return nil
}

func sanitizePath(params *RunParams) error {
	if params.Path == "" {
		var wd string
		wd, err := osutils.Getwd()
		if err != nil {
			return err
		}

		params.Path = wd
	}

	return nil
}

// Run kicks-off the runner.
func (r *Initialize) Run(params *RunParams) error {
	_, err := run(params, r.config, r.out)
	return err
}

func run(params *RunParams, config projectfile.ConfigGetter, out output.Outputer) (string, error) {
	if err := params.Namespace.Validate(); err != nil {
		return "", locale.WrapInputError(err, "init_invalid_namespace_err", "The provided namespace argument is invalid.")
	}

	if err := sanitizePath(params); err != nil {
		return "", locale.WrapInputError(err, "err_init_sanitize_path", "Could not prepare path: {{.V0}}", err.Error())
	}

	proj, err := project.FromPath(params.Path)
	if err != nil {
		if !errs.Matches(err, &projectfile.ErrorNoProject{}) {
			return "", locale.WrapError(err, "err_init_project", "Could not parse project information.")
		}
		proj = nil
	}

	isHeadless := proj != nil && proj.IsHeadless()

	if _, err := fileutils.PrepareDir(params.Path); err != nil {
		return "", locale.WrapError(err, "err_init_preparedir", "Could not create directory at [NOTICE]{{.V0}}[/RESET]. Error: {{.V1}}", params.Path, err.Error())
	}

	logging.Debug("Init: %s/%s %v", params.Namespace.Owner, params.Namespace.Project, params.Private)

	if isHeadless {
		err = proj.Source().SetNamespace(params.Namespace.Owner, params.Namespace.Project)
		if err != nil {
			return "", locale.WrapError(err, "err_init_set_namespace", "Could not set namespace in project file")
		}
	} else {
		// Sanitize rest of params
		if err := sanitize(params, config); err != nil {
			return "", err
		}

		createParams := &projectfile.CreateParams{
			Owner:     params.Namespace.Owner,
			Project:   params.Namespace.Project,
			Language:  params.language.String(),
			Directory: params.Path,
			Private:   params.Private,
		}

		pjfile, err := projectfile.Create(createParams)
		if err != nil {
			return "", locale.WrapError(err, "err_init_pjfile", "Could not create project file")
		}
		if proj, err = project.New(pjfile, out); err != nil {
			return "", err
		}
	}

	err = params.language.Validate()
	if err != nil {
		return "", locale.WrapError(err, "err_init_lang", "Invalid language for project creation")
	}

	version := deriveVersion(params.language.Language, params.version)
	commitID, err := model.CommitInitial(model.HostPlatform, params.language.Requirement(), version)
	if err != nil {
		return "", locale.WrapError(err, "err_init_commit", "Could not create initial commit")
	}

	if err := proj.SetCommit(commitID.String()); err != nil {
		return "", locale.WrapError(err, "err_init_setcommit", "Could not store commit to project file")
	}

	logging.Debug("Creating Platform project and pushing it")

	platformProject, err := model.CreateEmptyProject(params.Namespace.Owner, params.Namespace.Project, params.Private)
	if err != nil {
		return "", locale.WrapInputError(err, "err_init_create_project", "Failed to create a Platform project at {{.V0}}.", params.Namespace.String())
	}

	branch, err := model.DefaultBranchForProject(platformProject) // only one branch for newly created project
	if err != nil {
		return "", locale.NewInputError("err_no_default_branch")
	}

	err = model.UpdateProjectBranchCommitWithModel(platformProject, branch.Label, commitID)
	if err != nil {
		return "", locale.WrapError(err, "err_init_push", "Failed to push to the newly created Platform project at {{.V0}}", params.Namespace.String())
	}

	projectfile.StoreProjectMapping(config, params.Namespace.String(), filepath.Dir(proj.Source().Path()))

	out.Notice(locale.Tr(
		"init_success",
		params.Namespace.Owner,
		params.Namespace.Project,
		params.Path,
	))

	return params.Path, nil
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
