package projectfile

import (
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/ActiveState/ActiveState-CLI/internal/constants"
	"github.com/ActiveState/ActiveState-CLI/internal/failures"
	"github.com/ActiveState/ActiveState-CLI/internal/logging"
	yaml "gopkg.in/yaml.v2"
)

// Project covers the top level project structure of our yaml
type Project struct {
	Name         string     `yaml:"name"`
	Owner        string     `yaml:"owner"`
	Version      string     `yaml:"version"`
	Environments string     `yaml:"environments"`
	Platforms    []Platform `yaml:"platforms"`
	Languages    []Language `yaml:"languages"`
	Variables    []Variable `yaml:"variables"`
	Hooks        []Hook     `yaml:"hooks"`
	Commands     []Command  `yaml:"commands"`
	path         string     // "private"
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

// Hook covers the hook structure, which goes under Project
type Hook struct {
	Name        string     `yaml:"name"`
	Value       string     `yaml:"value"`
	Constraints Constraint `yaml:"constraints"`
}

// Command covers the command structure, which goes under Project
type Command struct {
	Name        string     `yaml:"name"`
	Value       string     `yaml:"value"`
	Constraints Constraint `yaml:"constraints"`
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

	return &project, err
}

// Path returns the project's activestate.yaml file path.
func (p *Project) Path() string {
	return p.path
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

// Get the project configuration
func Get() (*Project, error) {
	if persistentProject != nil {
		return persistentProject, nil
	}
	projectFilePath := os.Getenv(constants.ProjectEnvVarName)
	if projectFilePath == "" {
		projectFilePath = getProjectFilePath()
	}
	_, err := ioutil.ReadFile(projectFilePath)
	if err != nil {
		logging.Warning("Cannot load config file: %v", err)
		return nil, failures.App.New("Cannot load config. Make sure your config file is in the project root")
	}
	return Parse(projectFilePath)
}

// Persist "activates" the given project and makes it such that subsequent calls
// to Get() return this project.
// Only one project can persist at a time.
func (p *Project) Persist() {
	persistentProject = p
	os.Setenv(constants.ProjectEnvVarName, p.Path())
}
