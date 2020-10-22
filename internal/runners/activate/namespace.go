package activate

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/osutils"
	"github.com/ActiveState/cli/internal/prompt"
	"github.com/ActiveState/cli/pkg/project"

	"github.com/thoas/go-funk"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/locale"
)

type configAble interface {
	Set(key string, value interface{})
	GetString(key string) string
	GetStringSlice(key string) []string
}

// NamespaceSelect will select the right directory associated with a namespace, and chdir into it
type NamespaceSelect struct {
	config   configAble
	prompter prompt.Prompter
}

func NewNamespaceSelect(config configAble, prime primeable) *NamespaceSelect {
	return &NamespaceSelect{config, prime.Prompt()}
}

func (r *NamespaceSelect) Run(namespace string, preferredPath string) (string, error) {
	// Detect targetPath either by preferredPath or by prompting the user
	targetPath := preferredPath
	if targetPath == "" {
		var err error
		targetPath, err = r.promptForPath(namespace)
		if err != nil {
			return "", err
		}
	}

	// Validate that target path doesn't contain a config for a different namespace
	if fail := r.validatePath(namespace, targetPath); fail != nil {
		return "", fail
	}

	// Save path for future use
	key := fmt.Sprintf("project_%s", namespace)
	paths := r.availablePaths(namespace)
	paths = append(paths, targetPath)
	r.config.Set(key, paths)

	fail := fileutils.MkdirUnlessExists(targetPath)
	if fail != nil {
		return "", fail
	}

	return targetPath, nil
}

func (r *NamespaceSelect) promptForPath(namespace string) (string, error) {
	// If no targetPath was given try to get it from our config (ie. previous activations)
	paths := r.availablePaths(namespace)
	if len(paths) > 0 {
		targetPath, err := r.promptAvailablePaths(paths)
		if err != nil {
			return "", err
		}
		if targetPath != nil {
			return *targetPath, nil
		}
	}

	// Is targetPath STILL nil? Prompt the user for a path
	userPath, err := r.promptForPathInput(namespace)
	if err != nil {
		return "", err
	}

	return userPath, nil
}

// availablePaths returns any locations that this namespace is used, it strips out
// duplicates and paths that are no longer valid
func (r *NamespaceSelect) availablePaths(namespace string) []string {
	key := fmt.Sprintf("project_%s", namespace)
	paths := r.config.GetStringSlice(key)
	paths = funk.FilterString(paths, func(path string) bool {
		return fileutils.FileExists(filepath.Join(path, constants.ConfigFileName))
	})
	paths = funk.UniqString(paths)
	r.config.Set(key, paths)
	return paths
}

func (r *NamespaceSelect) promptAvailablePaths(paths []string) (*string, *failures.Failure) {
	if len(paths) == 0 {
		return nil, nil
	}

	noneStr := locale.T("activate_select_optout")
	choices := append(paths, noneStr)
	path, fail := r.prompter.Select(locale.Tl("activate_existing_title", "Existing Checkout"), locale.T("activate_namespace_existing"), choices, "")
	if fail != nil {
		return nil, fail
	}
	if path != "" && path != noneStr {
		return &path, nil
	}

	return nil, nil
}

// promptForPathInput will prompt the user for a location to save the project at
func (r *NamespaceSelect) promptForPathInput(namespace string) (string, *failures.Failure) {
	wd, err := getSafeWorkDir()
	if err != nil {
		return "", failures.FailRuntime.Wrap(err)
	}

	directory, fail := r.prompter.Input(
		locale.Tl("choose_dest", "Choose Destination"),
		locale.Tr("activate_namespace_location", namespace), filepath.Join(wd, namespace), prompt.InputRequired)
	if fail != nil {
		return "", fail
	}
	if directory == "" {
		return "", failures.FailUserInput.New("err_must_provide_directory")
	}

	logging.Debug("Using: %s", directory)

	return directory, nil
}

func (r *NamespaceSelect) validatePath(namespace string, path string) *failures.Failure {
	configFile := filepath.Join(path, constants.ConfigFileName)
	if !fileutils.FileExists(configFile) {
		return nil
	}

	pj, fail := project.Parse(configFile)
	if fail != nil {
		return fail
	}

	pjns := fmt.Sprintf("%s/%s", pj.Owner(), pj.Name())

	if pjns != namespace {
		return failures.FailUserInput.New("err_target_path_namespace_match", namespace, pjns)
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
