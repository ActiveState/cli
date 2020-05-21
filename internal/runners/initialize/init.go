package initialize

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/language"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/osutils"
	"github.com/ActiveState/cli/internal/print"
	"github.com/ActiveState/cli/pkg/project"
	"github.com/ActiveState/cli/pkg/projectfile"
)

// RunParams stores run func parameters.
type RunParams struct {
	Namespace *project.Namespaced
	Path      string
	Style     string
	Language  string
	language  language.Language
	version   string
}

// Initialize stores scope-related dependencies.
type Initialize struct {
}

func prepare(params *RunParams) error {
	if params.Language == "" {
		// Manually check for language requirement, because we need to fallback on the --language flag to support editor.V0
		return failures.FailUserInput.New(locale.T("err_init_no_language"))
	}
	langParts := strings.Split(params.Language, "@")
	if len(langParts) > 1 {
		params.version = langParts[1]
	}

	params.language = language.MakeByName(langParts[0])
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
		return fail
	}

	if params.Path == "" {
		var wd string
		wd, err := osutils.Getwd()
		if err != nil {
			return err
		}

		wd, err = fileutils.PrepareDir(wd)
		if err != nil {
			return err
		}

		empty, fail := fileutils.IsEmptyDir(wd)
		if fail != nil {
			return fail
		}

		params.Path = wd
		if !empty {
			params.Path = filepath.Join(wd, fmt.Sprintf(
				"%s/%s", params.Namespace.Owner, params.Namespace.Project,
			))
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

// New returns a prepared ptr to Initialize instance.
func New() *Initialize {
	return &Initialize{}
}

// Run kicks-off the runner.
func (r *Initialize) Run(params *RunParams) error {
	_, err := run(params)
	return err
}

func run(params *RunParams) (string, error) {
	if err := prepare(params); err != nil {
		return "", err
	}

	logging.Debug("Init: %s/%s", params.Namespace.Owner, params.Namespace.Project)

	createParams := &projectfile.CreateParams{
		Owner:           params.Namespace.Owner,
		Project:         params.Namespace.Project,
		Language:        params.language.String(),
		LanguageVersion: params.version,
		Directory:       params.Path,
	}

	if params.Style == SkeletonEditor {
		createParams.Content = locale.T("editor_yaml")
	}

	fail := projectfile.Create(createParams)
	if fail != nil {
		return "", fail
	}

	print.Line(locale.Tr(
		"init_success",
		params.Namespace.Owner,
		params.Namespace.Project,
		params.Path,
	))

	return params.Path, nil
}
