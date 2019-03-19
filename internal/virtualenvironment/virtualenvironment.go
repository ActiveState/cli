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

var persisted *VirtualEnvironment

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

type VirtualEnvironment struct {
	venvs               map[string]VirtualEnvironmenter
	onDownloadArtifacts func()
	onInstallArtifacts  func()
}

// Get returns a persisted version of VirtualEnvironment{}
func Get() *VirtualEnvironment {
	if persisted == nil {
		persisted = Init()
	}
	return persisted
}

// Init creates an instance of VirtualEnvironment{} with default settings
func Init() *VirtualEnvironment {
	return &VirtualEnvironment{venvs: make(map[string]VirtualEnvironmenter)}
}

// Activate the virtual environment
func (v *VirtualEnvironment) Activate() *failures.Failure {
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
		logging.Debug("Activating Virtual Environment: %+q", lang.ID())
		if _, failure := v.activateLanguage(lang); failure != nil {
			return failure
		}
	}

	return nil
}

func (v *VirtualEnvironment) OnDownloadArtifacts(f func()) { v.onDownloadArtifacts = f }

func (v *VirtualEnvironment) OnInstallArtifacts(f func()) { v.onInstallArtifacts = f }

// activateLanguage returns an environment for the given language, this will activate the
// virtual directory structure and set up the necessary environment variables if the venv
// wasnt already initialized, otherwise it will just return the venv.
func (v *VirtualEnvironment) activateLanguage(lang *project.Language) (VirtualEnvironmenter, *failures.Failure) {
	if venv, ok := v.venvs[lang.ID()]; ok {
		return venv, nil
	}

	hashedLangSpace := shortHash(lang.Source().Owner + "-" + lang.Source().Name + "-" + lang.ID())
	cacheDir := path.Join(config.GetCacheDir(), hashedLangSpace)

	var venv VirtualEnvironmenter
	var failure *failures.Failure

	switch strings.ToLower(lang.Name()) {
	case "python", "python3":
		rtInstaller, failure := python.NewInstaller(cacheDir)
		if failure != nil {
			return nil, failure
		}

		rtInstaller.OnDownload(v.onDownloadArtifacts)
		rtInstaller.OnInstall(v.onInstallArtifacts)

		venv, failure = python.NewVirtualEnvironment(cacheDir, rtInstaller)
		if failure != nil {
			return nil, failure
		}
	default:
		return nil, failures.FailUser.New(locale.Tr("warning_language_not_yet_supported", lang.Name()))
	}

	logging.Debug("Activating Virtual Environment: %s", venv.Language())
	if failure = venv.Activate(); failure != nil {
		return nil, failure
	}

	v.venvs[lang.ID()] = venv
	return venv, nil
}

// GetEnv returns a map of the cumulative environment variables for all active virtual environments
func (v *VirtualEnvironment) GetEnv() map[string]string {
	env := map[string]string{}

	for _, venv := range v.venvs {
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

	// Avoid leaking project
	if funk.Contains(env, constants.ProjectEnvVarName) {
		delete(env, constants.ProjectEnvVarName)
	}

	pjfile := projectfile.Get()
	env[constants.ActivatedStateEnvVarName] = filepath.Dir(pjfile.Path())

	return env
}

// WorkingDirectory returns the working directory to use for the current environment
func (v *VirtualEnvironment) WorkingDirectory() string {
	for _, venv := range v.venvs {
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
