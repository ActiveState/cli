package projectfile

import (
	"io/ioutil"
	"os"
	"regexp"
	"strings"

	"gopkg.in/yaml.v2"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
)

var (
	// FailNoProject identifies a failure as being due to a missing project file
	FailNoProject = failures.Type("projectfile.fail.noproject")

	// FailParseProject identifies a failure as being due inability to parse file contents
	FailParseProject = failures.Type("projectfile.fail.parseproject")

	// FailValidate identifies a failure during validation
	FailValidate = failures.Type("projectfile.fail.validate")

	// FailInvalidVersion identifies a failure as being due to an invalid version format
	FailInvalidVersion = failures.Type("projectfile.fail.version")
)

// Version is used in cases where we only care about parsing the version field. In all other cases the version is parsed via
// the Project struct
type Version struct {
	Version string `yaml:"version"`
}

// ProjectSimple reflects a bare basic project structure
type ProjectSimple struct {
	Name  string `yaml:"name"`
	Owner string `yaml:"owner"`
}

// Project covers the top level project structure of our yaml
type Project struct {
	Name         string      `yaml:"name"`
	Owner        string      `yaml:"owner"`
	Namespace    string      `yaml:"namespace,omitempty"`
	Version      string      `yaml:"version,omitempty"`
	Environments string      `yaml:"environments,omitempty"`
	Platforms    []Platform  `yaml:"platforms,omitempty"`
	Languages    []Language  `yaml:"languages,omitempty"`
	Variables    []*Variable `yaml:"variables,omitempty"`
	Events       []Event     `yaml:"events,omitempty"`
	Scripts      []Script    `yaml:"scripts,omitempty"`
	path         string      // "private"
}

// Platform covers the platform structure of our yaml
type Platform struct {
	Name         string `yaml:"name,omitempty"`
	Os           string `yaml:"os,omitempty"`
	Version      string `yaml:"version,omitempty"`
	Architecture string `yaml:"architecture,omitempty"`
	Libc         string `yaml:"libc,omitempty"`
	Compiler     string `yaml:"compiler,omitempty"`
}

// Build covers the build map, which can go under languages or packages
// Build can hold variable keys, so we cannot predict what they are, hence why it is a map
type Build map[string]string

// Language covers the language structure, which goes under Project
type Language struct {
	Name        string     `yaml:"name"`
	Version     string     `yaml:"version,omitempty"`
	Constraints Constraint `yaml:"constraints,omitempty"`
	Build       Build      `yaml:"build,omitempty"`
	Packages    []Package  `yaml:"packages,omitempty"`
}

// Constraint covers the constraint structure, which can go under almost any other struct
type Constraint struct {
	Platform    string `yaml:"platform,omitempty"`
	Environment string `yaml:"environment,omitempty"`
}

// Package covers the package structure, which goes under the language struct
type Package struct {
	Name        string     `yaml:"name"`
	Version     string     `yaml:"version"`
	Constraints Constraint `yaml:"constraints,omitempty"`
	Build       Build      `yaml:"build,omitempty"`
}

// Event covers the event structure, which goes under Project
type Event struct {
	Name        string     `yaml:"name"`
	Value       string     `yaml:"value"`
	Constraints Constraint `yaml:"constraints,omitempty"`
}

// Script covers the script structure, which goes under Project
type Script struct {
	Name        string     `yaml:"name"`
	Value       string     `yaml:"value"`
	Standalone  bool       `yaml:"standalone,omitempty"`
	Constraints Constraint `yaml:"constraints,omitempty"`
}

var persistentProject *Project

// Parse the given filepath, which should be the full path to an activestate.yaml file
func Parse(filepath string) (*Project, *failures.Failure) {
	dat, err := ioutil.ReadFile(filepath)
	if err != nil {
		return nil, failures.FailIO.Wrap(err)
	}

	project := Project{}
	err = yaml.Unmarshal([]byte(dat), &project)
	project.path = filepath

	if err != nil {
		return nil, FailNoProject.New(locale.T("err_project_parse", map[string]interface{}{"Error": err.Error()}))
	}

	return &project, project.Parse()
}

// Parse further processes the current file by parsing mixed values (something go-yaml doesnt handle)
func (p *Project) Parse() *failures.Failure {
	for _, variable := range p.Variables {
		fail := variable.Parse()
		if fail != nil {
			return fail
		}
	}

	return nil
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
func (p *Project) Save() *failures.Failure {
	dat, err := yaml.Marshal(p)
	if err != nil {
		return failures.FailMarshal.Wrap(err)
	}

	f, err := os.Create(p.Path())
	if err != nil {
		return failures.FailIO.Wrap(err)
	}
	defer f.Close()

	_, err = f.Write([]byte(dat))
	if err != nil {
		return failures.FailIO.Wrap(err)
	}

	return nil
}

// Returns the path to the project activestate.yaml
func getProjectFilePath() (string, *failures.Failure) {
	projectFilePath := os.Getenv(constants.ProjectEnvVarName)
	if projectFilePath != "" {
		return projectFilePath, nil
	}

	root, err := os.Getwd()
	if err != nil {
		logging.Warning("Could not get project root path: %v", err)
		return "", failures.FailOS.Wrap(err)
	}
	return fileutils.FindFileInPath(root, constants.ConfigFileName)
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

	// we do not want to use a path provided by state if we're running tests
	projectFilePath, failure := getProjectFilePath()
	if failure != nil {
		return nil, failure
	}

	_, err := ioutil.ReadFile(projectFilePath)
	if err != nil {
		logging.Warning("Cannot load config file: %v", err)
		return nil, FailNoProject.New(locale.T("err_no_projectfile"))
	}
	project, fail := Parse(projectFilePath)
	if fail != nil {
		return nil, FailParseProject.New(locale.Tr("err_parse_project", fail.Error()))
	}

	if project.Name == "" || project.Owner == "" {
		return nil, FailValidate.New("err_invalid_project_name_owner")
	}

	project.Persist()
	return project, nil
}

// ParseVersion parses the version field from the projectfile, and ONLY the version field. This is to ensure it doesn't
// trip over older activestate.yaml's with breaking changes
func ParseVersion() (string, *failures.Failure) {
	var projectFilePath string
	if persistentProject != nil {
		projectFilePath = persistentProject.Path()
	} else {
		var fail *failures.Failure
		projectFilePath, fail = getProjectFilePath()
		if fail != nil {
			// Not being able to find a project file is not a failure for the purposes of this function
			return "", nil
		}
	}

	if projectFilePath == "" {
		return "", nil
	}

	dat, err := ioutil.ReadFile(projectFilePath)
	if err != nil {
		return "", failures.FailIO.Wrap(err)
	}

	versionStruct := Version{}
	err = yaml.Unmarshal([]byte(dat), &versionStruct)
	if err != nil {
		return "", FailParseProject.Wrap(err)
	}

	if versionStruct.Version == "" {
		return "", nil
	}

	version := strings.TrimSpace(versionStruct.Version)
	match, fail := regexp.MatchString("^\\d+\\.\\d+\\.\\d+-\\d+$", version)
	if fail != nil || !match {
		return "", FailInvalidVersion.New(locale.T("err_invalid_version"))
	}

	return versionStruct.Version, nil
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
	if p.Name == "" || p.Owner == "" {
		failures.Handle(failures.FailDeveloper.New("err_persist_invalid_project"), locale.T("err_invalid_project_name_owner"))
		os.Exit(1)
	}
	persistentProject = p
	os.Setenv(constants.ProjectEnvVarName, p.Path())
}
