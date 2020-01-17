package initialize

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/language"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/print"
	"github.com/ActiveState/cli/internal/runners/initialize/skeleton"
	"github.com/ActiveState/cli/pkg/project"
	"github.com/ActiveState/cli/pkg/projectfile"
)

type setter interface {
	Set(key string, value interface{})
}

// RunParams stores run func parameters.
type RunParams struct {
	Namespace *project.Namespace
	Path      string
	Style     skeleton.Style
	Language  language.Language
}

// Initialize stores scope-related dependencies.
type Initialize struct {
	config setter
}

// New returns a prepared ptr to Initialize instance.
func New(config setter) *Initialize {
	return &Initialize{config}
}

// Run kicks-off the runner.
func (r *Initialize) Run(params *RunParams) error {
	_, err := run(r.config, params)
	return err
}

func run(config setter, params *RunParams) (string, error) {
	if err := prepare(params); err != nil {
		return "", err
	}

	logging.Debug("Init: %s/%s", params.Namespace.Owner, params.Namespace.Project)

	if params.Language.Recognized() && params.Language.Executable().Available() {
		// Store language for when we run 'state push'
		config.Set(params.Path+"_language", params.Language)
	}

	createParams := &projectfile.CreateParams{
		Owner:     params.Namespace.Owner,
		Project:   params.Namespace.Project,
		Directory: params.Path,
	}
	if params.Style == skeleton.Editor {
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

func prepare(params *RunParams) error {
	// Fail if target dir already has an activestate.yaml
	if fileutils.FileExists(filepath.Join(params.Path, constants.ConfigFileName)) {
		absPath, err := filepath.Abs(params.Path)
		if err != nil {
			return failures.FailIO.Wrap(err)
		}
		return failures.FailUserInput.New("err_init_file_exists", absPath)
	}

	if params.Style == skeleton.Unknown {
		return failures.FailUserInput.New("err_init_invalid_skeleton_style")
	}

	if params.Namespace == nil {
		return failures.FailUserInput.New("err_init_must_provide_namespace")
	}
	if params.Namespace.Owner == "" {
		return failures.FailUserInput.New("err_init_owner_missing")
	}
	if params.Namespace.Project == "" {
		return failures.FailUserInput.New("err_init_project_missing")
	}

	if params.Path == "" {
		wd, err := os.Getwd()
		if err != nil {
			return err
		}
		params.Path = filepath.Join(wd, fmt.Sprintf(
			"%s/%s", params.Namespace.Owner, params.Namespace.Project,
		))
	}

	return nil
}
