package projectfile

import (
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/mitchellh/hashstructure"
	yaml "gopkg.in/yaml.v2"
)

// FailNoProject identifies a failure as being due to a missing project file
var FailNoProject = failures.Type("projectfile.fail.noproject")

// FailParseProject identifies a failure as being due inability to parse file contents
var FailParseProject = failures.Type("projectfile.fail.parseproject")

// Project covers the top level project structure of our yaml
type Project struct {
	Name         string      `yaml:"name"`
	Owner        string      `yaml:"owner"`
	Namespace    string      `yaml:"namespace"`
	Version      string      `yaml:"version"`
	Environments string      `yaml:"environments"`
	Platforms    []Platform  `yaml:"platforms"`
	Languages    []Language  `yaml:"languages"`
	Variables    []Variable  `yaml:"variables"`
	Hooks        []Hook      `yaml:"hooks"`
	Commands     []Command   `yaml:"commands"`
	Secrets      SecretSpecs `yaml:"secrets"`
	path         string      // "private"
}

// Platform covers the platform structure of our yaml
type Platform struct {
	Name         string `yaml:"name"`
	Os           string `yaml:"os"`
	Version      string `yaml:"version"`
	Architecture string `yaml:"architecture"`
	Libc         string `yaml:"libc"`
	Compiler     string `yaml:"compiler"`
}

// Build covers the build map, which can go under languages or packages
// Build can hold variable keys, so we cannot predict what they are, hence why it is a map
type Build map[string]string

// Language covers the language structure, which goes under Project
type Language struct {
	Name        string     `yaml:"name"`
	Version     string     `yaml:"version"`
	Constraints Constraint `yaml:"constraints"`
	Build       Build      `yaml:"build"`
	Packages    []Package  `yaml:"packages"`
}

// Constraint covers the constraint structure, which can go under almost any other struct
type Constraint struct {
	Platform    string `yaml:"platform"`
	Environment string `yaml:"environment"`
}

// Package covers the package structure, which goes under the language struct
type Package struct {
	Name        string     `yaml:"name"`
	Version     string     `yaml:"version"`
	Constraints Constraint `yaml:"constraints"`
	Build       Build      `yaml:"build"`
}

// Variable covers the variable structure, which goes under Project
type Variable struct {
	Name        string     `yaml:"name"`
	Value       string     `yaml:"value"`
	Constraints Constraint `yaml:"constraints"`
}

// Hash return a hashed version of the variable
func (v *Variable) Hash() (string, error) {
	hash, err := hashstructure.Hash(v, nil)
	if err != nil {
		logging.Errorf("Cannot hash variable: %v", err)
		return "", err
	}
	return fmt.Sprintf("%X", hash), nil
}

// Hook covers the hook structure, which goes under Project
type Hook struct {
	Name        string     `yaml:"name"`
	Value       string     `yaml:"value"`
	Constraints Constraint `yaml:"constraints"`
}

// Hash return a hashed version of the hook
func (h *Hook) Hash() (string, error) {
	hash, err := hashstructure.Hash(h, nil)
	if err != nil {
		logging.Errorf("Cannot hash hook: %v", err)
		return "", err
	}
	return fmt.Sprintf("%X", hash), nil
}

// Command covers the command structure, which goes under Project
type Command struct {
	Name        string     `yaml:"name"`
	Value       string     `yaml:"value"`
	Standalone  bool       `yaml:"standalone"`
	Constraints Constraint `yaml:"constraints"`
}

// SecretSpec covers the secret specification structure, which goes under Project
type SecretSpec struct {
	Name      string `yaml:"name"`
	IsProject bool   `yaml:"project"`
	IsUser    bool   `yaml:"user"`
}

// Scope returns a human readable representation of the scope of this Secret.
func (spec SecretSpec) Scope() string {
	if spec.IsUser && spec.IsProject {
		return locale.T("secrets_scope_user_project")
	} else if spec.IsUser {
		return locale.T("secrets_scope_user_org")
	} else if spec.IsProject {
		return locale.T("secrets_scope_project")
	}
	return locale.T("secrets_scope_org")
}

// SecretSpecs adds functionality around slices of SecretSpecs.
type SecretSpecs []*SecretSpec

// GetByName find the SecretSpec with the requested name.
func (specs SecretSpecs) GetByName(specName string) *SecretSpec {
	for _, spec := range specs {
		if strings.EqualFold(specName, spec.Name) {
			return spec
		}
	}
	return nil
}

var persistentProject *Project

// Parse the given filepath, which should be the full path to an activestate.yaml file
func Parse(filepath string) (*Project, error) {
	dat, err := ioutil.ReadFile(filepath)
	if err != nil {
		return nil, err
	}

	project := Project{}
	err = yaml.Unmarshal([]byte(dat), &project)
	project.path = filepath

	if err != nil {
		return nil, FailNoProject.New(locale.T("err_project_parse", map[string]interface{}{"Error": err.Error()}))
	}

	return &project, err
}

// Path returns the project's activestate.yaml file path.
func (p *Project) Path() string {
	return p.path
}

// SetPath sets the path of the project file and should generally only be used by tests
func (p *Project) SetPath(path string) {
	p.path = path
}

// Save the project to its activestate.yaml file
func (p *Project) Save() error {
	dat, err := yaml.Marshal(p)
	if err != nil {
		return err
	}

	f, err := os.Create(p.Path())
	defer f.Close()
	if err != nil {
		return err
	}

	_, err = f.Write([]byte(dat))
	if err != nil {
		return err
	}

	return nil
}

// Returns the path to the project activestate.yaml
func getProjectFilePath() string {
	root, err := os.Getwd()
	if err != nil {
		logging.Warning("Could not get project root path: %v", err)
		return ""
	}
	return filepath.Join(root, constants.ConfigFileName)
}

// Get returns the project configration in an unsafe manner (exits if errors occur)
func Get() *Project {
	project, err := GetSafe()
	if err != nil {
		failures.Handle(err, locale.T("err_project_file_unavailable"))
		os.Exit(1)
	}

	return project
}

// GetSafe returns the project configuration in a safe manner (returns error)
func GetSafe() (*Project, *failures.Failure) {
	if persistentProject != nil {
		return persistentProject, nil
	}

	projectFilePath := os.Getenv(constants.ProjectEnvVarName)
	// we do not want to use a path provided by state if we're running tests
	if projectFilePath == "" || flag.Lookup("test.v") != nil {
		projectFilePath = getProjectFilePath()
	}

	_, err := ioutil.ReadFile(projectFilePath)
	if err != nil {
		logging.Warning("Cannot load config file: %v", err)
		return nil, FailNoProject.New(locale.T("err_no_projectfile"))
	}
	project, err := Parse(projectFilePath)
	if err != nil {
		return nil, FailParseProject.New(locale.T("err_parse_project"))
	}
	project.Persist()
	return project, nil
}

// Reset the current state, which unsets the persistent project
func Reset() {
	persistentProject = nil
	os.Unsetenv(constants.ProjectEnvVarName)
}

// Persist "activates" the given project and makes it such that subsequent calls
// to Get() return this project.
// Only one project can persist at a time.
func (p *Project) Persist() {
	persistentProject = p
	os.Setenv(constants.ProjectEnvVarName, p.Path())
}
