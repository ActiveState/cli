package initialize

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/ActiveState/cli/pkg/project"

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

type Init struct {
	config configAble
}

type Options struct {
	Namespace,
	Path,
	Language,
	Skeleton string
}

const (
	Base   = "base"
	Editor = "editor"
)

func NewInit(config configAble) *Init {
	return &Init{config}
}

func (r *Init) Run(opts Options) error {
	_, err := r.run(opts)
	return err
}

func (r *Init) run(opts Options) (string, error) {
	if opts.Namespace == "" {
		return "", failures.FailUserInput.New("err_init_must_provide_namespace")
	}

	ns, fail := project.ParseNamespace(opts.Namespace)
	if fail != nil {
		return "", fail
	}

	// Detect path if none was provided
	if opts.Path == "" {
		wd, err := os.Getwd()
		if err != nil {
			return "", err
		}
		opts.Path = filepath.Join(wd, opts.Namespace)
	}

	// Fail if target dir already has an activestate.yaml
	if fileutils.FileExists(filepath.Join(opts.Path, constants.ConfigFileName)) {
		absPath, err := filepath.Abs(opts.Path)
		if err != nil {
			return "", failures.FailIO.Wrap(err)
		}
		return "", failures.FailUserInput.New("err_init_file_exists", absPath)
	}

	if opts.Language != "" {
		lang := language.MakeByName(opts.Language)
		if lang == language.Unknown {
			return "", failures.FailUserInput.New("err_init_invalid_language", opts.Language, strings.Join(language.AvailableNames(), ", "))
		}
		// Store language for when we run 'state push'
		r.config.Set(opts.Path+"_language", opts.Language)
	}

	params := &projectfile.CreateParams{
		Owner:     ns.Owner,
		Project:   ns.Project,
		Directory: opts.Path,
	}
	if opts.Skeleton != "" {
		switch strings.ToLower(opts.Skeleton) {
		case Editor:
			// Set our own custom content
			params.Content = locale.T("editor_yaml")
		case Base:
			// Use the default content set in the projectfile.Create function
		default:
			return "", failures.FailUserInput.New("err_init_invalid_skeleton_flag")
		}
	}

	// Create the activestate.yaml
	fail = projectfile.Create(params)
	if fail != nil {
		return "", fail
	}

	print.Line(locale.Tr("init_success", opts.Namespace, opts.Path))

	return opts.Path, nil
}
