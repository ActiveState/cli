package initialize

import (
	"os"
	"path/filepath"

	"github.com/ActiveState/cli/pkg/project"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/fileutils"
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

func NewInit(config configAble) *Init {
	return &Init{config}
}

func (r *Init) Run(namespace, path, language string) error {
	_, err := r.run(namespace, path, language)
	return err
}

func (r *Init) run(namespace, path, language string) (string, error) {
	if namespace == "" {
		return "", failures.FailUserInput.New("err_init_must_provide_namespace")
	}

	ns, fail := project.ParseNamespace(namespace)
	if fail != nil {
		return "", fail
	}

	// Detect path if none was provided
	if path == "" {
		wd, err := os.Getwd()
		if err != nil {
			return "", err
		}
		path = filepath.Join(wd, namespace)
	}

	// Fail if target dir already has an activestate.yaml
	if fileutils.FileExists(filepath.Join(path, constants.ConfigFileName)) {
		absPath, err := filepath.Abs(path)
		if err != nil {
			return "", failures.FailIO.Wrap(err)
		}
		return "", failures.FailUserInput.New("err_init_file_exists", absPath)
	}

	// Create the activestate.yaml
	if fail := projectfile.Create(ns.Owner, ns.Project, nil, path); fail != nil {
		return "", fail
	}

	// Store language for when we run 'state push'
	if language != "" {
		r.config.Set(path+"_language", language)
	}

	print.Line(locale.Tr("init_success", namespace, path))

	return path, nil
}
