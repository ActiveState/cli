package activate

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/ActiveState/cli/internal/config"
	"github.com/ActiveState/cli/internal/osutils"
	"github.com/ActiveState/cli/internal/prompt"
	"github.com/ActiveState/cli/pkg/project"
	"github.com/ActiveState/cli/pkg/projectfile"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/locale"
)

// NamespaceSelect will select the right directory associated with a namespace, and chdir into it
type NamespaceSelect struct {
	config   *config.Instance
	prompter prompt.Prompter
}

func NewNamespaceSelect(config *config.Instance, prime primeable) *NamespaceSelect {
	return &NamespaceSelect{config, prime.Prompt()}
}

func (r *NamespaceSelect) Run(namespace string, preferredPath string) (string, error) {
	// Detect targetPath either by preferredPath or by prompting the user
	targetPath := preferredPath
	if targetPath == "" {
		var err error
		targetPath, err = r.getProjectPath(namespace)
		if err != nil {
			return "", err
		}
	}

	// Validate that target path doesn't contain a config for a different namespace
	if err := r.validatePath(namespace, targetPath); err != nil {
		return "", err
	}

	err := fileutils.MkdirUnlessExists(targetPath)
	if err != nil {
		return "", err
	}

	return targetPath, nil
}

func (r *NamespaceSelect) getProjectPath(namespace string) (string, error) {
	// If no targetPath was given try to get it from our config (ie. previous activations)
	paths := projectfile.GetProjectPaths(r.config, namespace)
	if len(paths) > 0 {
		return paths[0], nil
	}

	targetPath, err := getSafeWorkDir()
	if err != nil {
		return "", locale.NewError("err_get_wd", "Could not get safe working directory")
	}

	return targetPath, nil
}

func (r *NamespaceSelect) validatePath(namespace string, path string) error {
	configFile := filepath.Join(path, constants.ConfigFileName)
	if !fileutils.FileExists(configFile) {
		return nil
	}

	pj, err := project.Parse(configFile)
	if err != nil {
		return err
	}

	pjns := fmt.Sprintf("%s/%s", pj.Owner(), pj.Name())
	if !pj.IsHeadless() && pjns != namespace {
		return locale.NewInputError("err_target_path_namespace_match", "", namespace, pjns)
	}

	return nil
}

func getSafeWorkDir() (string, error) {
	dir, err := osutils.Getwd()
	if err != nil {
		return "", err
	}

	if !strings.HasPrefix(strings.ToLower(dir), `c:\windows`) {
		return dir, nil
	}

	dir, err = os.UserHomeDir()
	if err != nil {
		return "", err
	}

	return dir, nil
}
