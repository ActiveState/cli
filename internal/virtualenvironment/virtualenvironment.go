package virtualenvironment

import (
	"crypto/sha1"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/ActiveState/cli/pkg/project"

	"github.com/ActiveState/cli/internal/config"
	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/virtualenvironment/python"
	"github.com/ActiveState/cli/pkg/projectfile"
	funk "github.com/thoas/go-funk"
)

// FailAlreadyActive is a failure given when a project is already active
var FailAlreadyActive = failures.Type("virtualenvironment.fail.alreadyactive", failures.FailUser)

// VirtualEnvironmenter defines the interface for our virtual environment packages, which should be contained in a sub-directory
// under the same directory as this file
type VirtualEnvironmenter interface {
	// Activate the given virtualenvironment
	Activate() *failures.Failure

	// Env returns the desired environment variables for this venv
	Env() map[string]string

	// Language returns the language name
	Language() string

	// WorkingDirectory returns the working directory for this venv, or an empty string if it has no preference
	WorkingDirectory() string

	// DataDir returns the configured data dir for this venv
	DataDir() string
}

type artifactHashable struct {
	Name    string
	Version string
	Build   map[string]string
}

var venvs = make(map[string]VirtualEnvironmenter)

// Activate the virtual environment
func Activate() *failures.Failure {
	logging.Debug("Activating Virtual Environment")

	activeProject := os.Getenv(constants.ActivatedStateEnvVarName)
	if activeProject != "" {
		return FailAlreadyActive.New("err_already_active")
	}

	project := project.Get()

	// expand project vars to environment vars
	for _, variable := range project.Variables() {
		val, failure := variable.Value()
		if failure != nil {
			return failure
		}
		os.Setenv(variable.Name(), val)
	}

	for _, lang := range project.Languages() {
		if _, failure := activateLanguage(lang); failure != nil {
			return failure
		}
	}

	return nil
}

// activateLanguage returns an environment for the given language, this will activate the
// virtual directory structure and set up the necessary environment variables if the venv
// wasnt already initialized, otherwise it will just return the venv.
func activateLanguage(lang *project.Language) (VirtualEnvironmenter, *failures.Failure) {
	if venv, ok := venvs[lang.ID()]; ok {
		return venv, nil
	}

	hashedLangSpace := shortHash(lang.Source().Owner + "-" + lang.Source().Name + "-" + lang.ID())
	cacheDir := path.Join(config.GetCacheDir(), hashedLangSpace)

	var venv VirtualEnvironmenter
	var failure *failures.Failure

	switch strings.ToLower(lang.Name()) {
	case "python", "python3":
		venv = python.NewVirtualEnvironment(cacheDir)
		failure = venv.Activate()
	default:
		return nil, failures.FailUser.New(locale.Tr("warning_language_not_yet_supported", lang.Name()))
	}

	if failure != nil {
		return nil, failure
	}

	venvs[lang.ID()] = venv
	return venv, nil
}

// GetEnv returns a map of the cumulative environment variables for all active virtual environments
func GetEnv() map[string]string {
	env := map[string]string{}

	for _, venv := range venvs {
		for k, v := range venv.Env() {
			if k == "PATH" && funk.Contains(env, "PATH") {
				env["PATH"] = v + string(os.PathListSeparator) + env["PATH"]
			} else {
				if funk.Contains(env, k) {
					logging.Warning("Two languages are defining the %s environment key, only one will be used", k)
				}
				env[k] = v
			}
		}
	}

	if funk.Contains(env, "PATH") {
		env["PATH"] = env["PATH"] + string(os.PathListSeparator) + os.Getenv("PATH")
	}

	pjfile := projectfile.Get()
	env[constants.ActivatedStateEnvVarName] = filepath.Dir(pjfile.Path())

	return env
}

// WorkingDirectory returns the working directory to use for the current environment
func WorkingDirectory() string {
	for _, venv := range venvs {
		wd := venv.WorkingDirectory()
		if wd != "" {
			return wd
		}
	}

	wd, err := os.Getwd()
	if err != nil {
		// Shouldn't happen unless something is seriously wrong with your system
		panic(locale.T("panic_couldnt_detect_wd", map[string]interface{}{"Error": err.Error()}))
	}

	return wd
}

// shortHash will return the first 4 bytes in base16 of the sha1 sum of the provided data.
//
// For example:
//   shortHash("ActiveState-TestProject-python2")
// 	 => e784c7e0
//
// This is useful for creating a shortened namespace for language installations.
func shortHash(data string) string {
	h := sha1.New()
	io.WriteString(h, data)
	return fmt.Sprintf("%x", h.Sum(nil)[:4])
}
