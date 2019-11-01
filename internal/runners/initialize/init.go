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

func NewInit(config configAble) *Init {
	return &Init{config}
}

func (r *Init) Run(namespace, path, langName string) error {
	_, err := r.run(namespace, path, langName)
	return err
}

func (r *Init) run(namespace, path, langName string) (string, error) {
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

	// Store language for when we run 'state push'
	if langName != "" {
		lang := language.MakeByName(langName)
		if lang == language.Unknown {
			return "", failures.FailUserInput.New("err_init_invalid_language", langName, strings.Join(language.AvailableNames(), ", "))
		}
		r.config.Set(path+"_language", langName)
	}

	// Create the activestate.yaml
	if fail := projectfile.Create(ns.Owner, ns.Project, nil, path); fail != nil {
		return "", fail
	}

	print.Line(locale.Tr("init_success", namespace, path))

	return path, nil
}
