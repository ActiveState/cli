package initialize

import (
	"path/filepath"
	"strings"

	"github.com/ActiveState/cli/pkg/platform/model"
	"github.com/gobuffalo/packr"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/language"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/osutils"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/primer"
	"github.com/ActiveState/cli/pkg/project"
	"github.com/ActiveState/cli/pkg/projectfile"
)

// RunParams stores run func parameters.
type RunParams struct {
	Namespace *project.Namespaced
	Path      string
	Style     string
	Language  string
	Private   bool
	language  language.Supported
	version   string
}

// Initialize stores scope-related dependencies.
type Initialize struct {
	out output.Outputer
}

type primeable interface {
	primer.Outputer
}

// New returns a prepared ptr to Initialize instance.
func New(prime primeable) *Initialize {
	return &Initialize{prime.Output()}
}

func sanitize(params *RunParams) error {
	if params.Language == "" {
		// Manually check for language requirement, because we need to fallback on the --language flag to support editor.V0
		return locale.NewInputError("err_init_no_language", "You need to supply the [NOTICE]language[/RESET] argument, run [ACTIONABLE]`state init --help`[/RESET] for more information.")
	}
	langParts := strings.Split(params.Language, "@")
	if len(langParts) > 1 {
		params.version = langParts[1]
	}

	params.language = language.Supported{language.MakeByName(langParts[0])}
	if !params.language.Recognized() {
		return language.NewUnrecognizedLanguageError(
			params.language.String(),
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

	if !styleRecognized(params.Style) {
		params.Style = SkeletonBase
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
	_, err := run(params, r.out)
	return err
}

func run(params *RunParams, out output.Outputer) (string, error) {
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
		if err := sanitize(params); err != nil {
			return "", err
		}

		createParams := &projectfile.CreateParams{
			Owner:     params.Namespace.Owner,
			Project:   params.Namespace.Project,
			Language:  params.language.String(),
			Directory: params.Path,
			Private:   params.Private,
		}

		if params.Style == SkeletonEditor {
			box := packr.NewBox("../../../assets/")
			createParams.Content = box.String("activestate.yaml.editor.tpl")
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

	version := params.version
	if version == "" {
		version = params.language.RecommendedVersion()
	}
	commitID, err := model.CommitInitial(model.HostPlatform, params.language.String(), version)
	if err != nil {
		return "", locale.WrapError(err, "err_init_commit", "Could not create initial commit")
	}

	if err := proj.SetCommit(commitID.String()); err != nil {
		return "", locale.WrapError(err, "err_init_setcommit", "Could not store commit to project file")
	}

	out.Notice(locale.Tr(
		"init_success",
		params.Namespace.Owner,
		params.Namespace.Project,
		params.Path,
	))

	return params.Path, nil
}
