package virtualenvironment

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/thoas/go-funk"

	"github.com/ActiveState/cli/internal/print"

	"github.com/ActiveState/cli/internal/artifact"
	"github.com/ActiveState/cli/internal/constraints"
	"github.com/ActiveState/cli/internal/fileutils"

	"github.com/ActiveState/cli/internal/distribution"
	"github.com/ActiveState/cli/internal/failures"

	"github.com/ActiveState/cli/internal/config"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/virtualenvironment/golang"
	"github.com/ActiveState/cli/internal/virtualenvironment/python"
	"github.com/ActiveState/cli/pkg/projectfile"
)

// VirtualEnvironmenter defines the interface for our virtual environment packages, which should be contained in a sub-directory
// under the same directory as this file
type VirtualEnvironmenter interface {
	// Activate the given virtualenvironment
	Activate() *failures.Failure

	// Env returns the desired environment variables for this venv
	Env() map[string]string

	// Language returns the language name
	Language() string

	// LanguageMeta holds the *projectfile.Language for this venv
	Artifact() *artifact.Artifact

	// SetLanguageMeta sets the language meta
	SetArtifact(*artifact.Artifact)

	// DataDir returns the configured data dir for this venv
	DataDir() string

	// SetDataDir sets the configured data for this venv
	SetDataDir(string)

	// LoadArtifact should load the given artifact into the venv
	LoadArtifact(*artifact.Artifact) *failures.Failure
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

	project := projectfile.Get()

	if project.Variables != nil {
		for _, variable := range project.Variables {
			if !constraints.IsConstrained(variable.Constraints) {
				os.Setenv(variable.Name, variable.Value)
			}
		}
	}

	dist, fail := distribution.Obtain()
	if fail != nil {
		return fail
	}

	// Load Languages
	print.Info(locale.T("info_activating_state", project))
	for _, artf := range dist.Languages {
		fail = createLanguageFolderStructure(artf)
		if fail != nil {
			return fail
		}

		env, fail := GetVenv(artf)
		if fail != nil {
			return fail
		}

		// Load language artifact
		fail = env.LoadArtifact(artf)
		if fail != nil {
			return fail
		}

		// Load Artifacts belonging to language
		for _, subArtf := range dist.Artifacts[artf.Hash] {
			fail := env.LoadArtifact(subArtf)
			if fail != nil {
				return fail
			}
		}
	}

	return nil
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

	return env
}

// GetVenv returns an environment for the given project and language, this will initialize the virtual directory structure
// and set up the necessary environment variables if the venv wasnt already initialized, otherwise it will just return
// the venv struct
func GetVenv(artf *artifact.Artifact) (VirtualEnvironmenter, *failures.Failure) {
	if _, ok := venvs[artf.Hash]; ok {
		return venvs[artf.Hash], nil
	}

	var venv VirtualEnvironmenter

	switch strings.ToLower(artf.Meta.Name) {
	case "python":
		venv = &python.VirtualEnvironment{}
		fail := ActivateLanguageVenv(artf, venv)

		if fail != nil {
			return nil, fail
		}
	case "go":
		venv = &golang.VirtualEnvironment{}
		fail := ActivateLanguageVenv(artf, venv)

		if fail != nil {
			return nil, fail
		}
	default:
		var T = locale.T
		return nil, failures.FailUser.New(T("warning_language_not_yet_supported", map[string]interface{}{
			"Language": artf.Meta.Name,
		}))
	}

	venvs[artf.Hash] = venv
	return venv, nil
}

// ActivateLanguageVenv activates the virtual environment for the given language
func ActivateLanguageVenv(artf *artifact.Artifact, venv VirtualEnvironmenter) *failures.Failure {
	project := projectfile.Get()
	datadir := config.GetDataDir()
	datadir = filepath.Join(datadir, "virtual", project.Owner, project.Name, artf.Meta.Name, artf.Meta.Version)

	err := os.RemoveAll(datadir)
	if err != nil {
		return failures.FailIO.Wrap(err)
	}

	venv.SetArtifact(artf)
	venv.SetDataDir(datadir)

	return venv.Activate()
}

func createLanguageFolderStructure(artf *artifact.Artifact) *failures.Failure {
	project := projectfile.Get()
	datadir := config.GetDataDir()

	if fail := fileutils.Mkdir(datadir, "packages"); fail != nil {
		return fail
	}

	os.RemoveAll(filepath.Join(datadir, "virtual", project.Owner, project.Name))

	fileutils.Mkdir(datadir, "virtual", project.Owner, project.Name, artf.Meta.Name, artf.Meta.Version)

	return nil
}
