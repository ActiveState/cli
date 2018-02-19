package projectfile

import (
	"crypto/sha1"
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

var currentProject *Project
var projectHash string

// configFilename from constants.ConfigFileName
var configFilename = constants.ConfigFileName

// Parse the given filepath, which should be the full path to an activestate.yaml file
func Parse(filepath string) (*Project, error) {
	dat, err := ioutil.ReadFile(filepath)
	if err != nil {
		return nil, err
	}

	project := Project{}
	err = yaml.Unmarshal([]byte(dat), &project)

	return &project, err
}

// Write to the given filepath, which should be the full path to an activestate.yaml file
func Write(filepath string, project *Project) error {
	dat, err := yaml.Marshal(&project)
	if err != nil {
		return err
	}

	f, err := os.Create(filepath)
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

func hashConfig(data []byte) string {
	hash := sha1.New()
	return string(hash.Sum(data))
}

// GetProjectFilePath returns the path to the project activestate.yaml
func GetProjectFilePath() string {
	root, err := os.Getwd()
	if err != nil {
		logging.Warning("Could not get project root path: %v", err)
		return ""
	}
	return filepath.Join(root, configFilename)
}

// Get the project configuration
// If no project file exists in the current directory and a project file was
// previously loaded (i.e. the state is activated), return the loaded project.
func Get() (*Project, error) {
	projectFilePath := GetProjectFilePath()
	if _, err := os.Stat(projectFilePath); err != nil && currentProject != nil {
		return currentProject, nil
	}
	data, err := ioutil.ReadFile(projectFilePath)
	hash := hashConfig(data)
	if err != nil {
		logging.Warning("Cannot load config file: %v", err)
		projectHash = ""
		return nil, failures.App.New("Cannot load config. Make sure your config file is in the project root")
	}
	if currentProject == nil || hash != projectHash {
		currentProject, err = Parse(projectFilePath)
		projectHash = hash
	}
	return currentProject, nil
}
