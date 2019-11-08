package initialize

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/language"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/print"
	"github.com/ActiveState/cli/pkg/projectfile"

	"github.com/ActiveState/cli/internal/failures"
)

type configAble interface {
	Set(key string, value interface{})
}

type SkeletonStyle string

const (
	Simple SkeletonStyle = ""
	Editor SkeletonStyle = "editor"
)

type Init struct {
	config configAble
}

type RunParams struct {
	Owner    string
	Project  string
	Path     string
	Skeleton SkeletonStyle
	Language language.Language
}

func (params *RunParams) Prepare() error {
	// Fail if target dir already has an activestate.yaml
	if fileutils.FileExists(filepath.Join(params.Path, constants.ConfigFileName)) {
		absPath, err := filepath.Abs(params.Path)
		if err != nil {
			return failures.FailIO.Wrap(err)
		}
		return failures.FailUserInput.New("err_init_file_exists", absPath)
	}

	if params.Owner == "" {
		return failures.FailUserInput.New("err_init_owner_missing")
	}
	if params.Project == "" {
		return failures.FailUserInput.New("err_init_project_missing")
	}

	if params.Path == "" {
		wd, err := os.Getwd()
		if err != nil {
			return err
		}
		params.Path = filepath.Join(wd, fmt.Sprintf("%s/%s", params.Owner, params.Project))
	}

	return nil
}

func NewInit(config configAble) *Init {
	return &Init{config}
}

func (r *Init) Run(params *RunParams) error {
	_, err := r.run(params)
	return err
}

func (r *Init) run(runParams *RunParams) (string, error) {
	err := runParams.Prepare()
	if err != nil {
		return "", err
	}

	if runParams.Language != language.Unknown {
		// Store language for when we run 'state push'
		r.config.Set(runParams.Path+"_language", runParams.Language)
	}

	createParams := &projectfile.CreateParams{
		Owner:     runParams.Owner,
		Project:   runParams.Project,
		Directory: runParams.Path,
	}

	if runParams.Skeleton == Editor {
		createParams.Content = locale.T("editor_yaml")
	}

	// Create the activestate.yaml
	fail := projectfile.Create(createParams)
	if fail != nil {
		return "", fail
	}

	print.Line(locale.Tr("init_success", runParams.Owner, runParams.Project, runParams.Path))

	return runParams.Path, nil
}
