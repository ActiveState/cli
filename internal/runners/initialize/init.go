package initialize

import (
	"path/filepath"
	"strings"

	"github.com/gobuffalo/packr"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/failures"
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
	out  output.Outputer
	proj *project.Project
}

type primeable interface {
	primer.Outputer
	primer.Projecter
}

// New returns a prepared ptr to Initialize instance.
func New(prime primeable) *Initialize {
	return &Initialize{prime.Output(), prime.Project()}
}

func prepare(params *RunParams) error {
	if params.Language == "" {
		// Manually check for language requirement, because we need to fallback on the --language flag to support editor.V0
		return locale.NewInputError("err_init_no_language", "You need to supply the [NOTICE]language[/RESET] argument, run `[ACTIONABLE]state init --help[/RESET]` for more information.")
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
			return failures.FailIO.Wrap(err)
		}
		return failures.FailUserInput.New("err_init_file_exists", absPath)
	}

	if !styleRecognized(params.Style) {
		params.Style = SkeletonBase
	}

	if fail := params.Namespace.Validate(); fail != nil {
		return locale.WrapInputError(fail.ToError(), "init_invalid_namespace_err", "The provided namespace argument is invalid.")
	}

	if params.Path == "" {
		var wd string
		wd, err := osutils.Getwd()
		if err != nil {
			return err
		}

		params.Path, err = fileutils.PrepareDir(wd)
		if err != nil {
			return err
		}
	} else {
		var err error
		params.Path, err = fileutils.PrepareDir(params.Path)
		if err != nil {
			return err
		}
	}

	return nil
}

// Run kicks-off the runner.
func (r *Initialize) Run(params *RunParams) error {
	_, err := run(params, r.out, r.proj)
	return err
}

func run(params *RunParams, out output.Outputer, proj *project.Project) (string, error) {
	// try to initialize project at --path if provided
	if params.Path != "" {
		var fail *failures.Failure
		proj, fail = project.Parse(filepath.Join(params.Path, constants.ConfigFileName))
		if fail != nil {
			logging.Debug("Could not parse project file at path: %v", fail.ToError())
			proj = nil
		}
	}

	path := params.Path
	// check if we are converting a headless commit
	convertHeadless := proj != nil && proj.IsHeadless()
	if convertHeadless {
		if params.Language != "" {
			return "", locale.NewInputError("init_headless_lang_provided_err", "Language argument cannot be provided when converting a headless commit.")
		}
	} else {
		if err := prepare(params); err != nil {
			return "", err
		}

		logging.Debug("Init: %s/%s %v", params.Namespace.Owner, params.Namespace.Project, params.Private)

		createParams := &projectfile.CreateParams{
			Owner:           params.Namespace.Owner,
			Project:         params.Namespace.Project,
			Language:        params.language.String(),
			LanguageVersion: params.version,
			Directory:       path,
			Private:         params.Private,
		}

		if params.Style == SkeletonEditor {
			box := packr.NewBox("../../../assets/")
			createParams.Content = box.String("activestate.yaml.editor.tpl")
		}

		fail := projectfile.Create(createParams)
		if fail != nil {
			return "", fail
		}
	}

	if convertHeadless {
		headlessCommitID := proj.CommitID()

		err := proj.Source().SetNamespace(params.Namespace.Owner, params.Namespace.Project)
		if err != nil {
			return "", errs.Wrap(err, "Could not set namespace in project file.")
		}

		fail := proj.Source().SetCommit(headlessCommitID, false)
		if fail != nil {
			return "", errs.Wrap(fail.ToError(), "Could not set commit id in project file.")
		}

		if path == "" {
			path, err = osutils.Getwd()
			if err != nil {
				return "", errs.Wrap(err, "Could not determine current working directory.")
			}
		}
	}

	out.Notice(locale.Tr(
		"init_success",
		params.Namespace.Owner,
		params.Namespace.Project,
		path,
	))

	return path, nil
}
