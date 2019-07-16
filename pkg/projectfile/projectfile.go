package projectfile

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"gopkg.in/yaml.v2"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/print"
)

var (
	// FailNoProject identifies a failure as being due to a missing project file
	FailNoProject = failures.Type("projectfile.fail.noproject")

	// FailParseProject identifies a failure as being due inability to parse file contents
	FailParseProject = failures.Type("projectfile.fail.parseproject", failures.FailUser)

	// FailValidate identifies a failure during validation
	FailValidate = failures.Type("projectfile.fail.validate")

	// FailInvalidVersion identifies a failure as being due to an invalid version format
	FailInvalidVersion = failures.Type("projectfile.fail.version")

	// FailSetCommitID identifies a failure as being caused by the commit id not getting set
	FailSetCommitID = failures.Type("projectfile.fail.setcommitid")

	// FailNewBlankPath identifies a failure as being caused by the commit id not getting set
	FailNewBlankPath = failures.Type("projectfile.fail.blanknewpath")

	// FailProjectExists identifies a failure as being caused by the commit id not getting set
	FailProjectExists = failures.Type("projectfile.fail.projectalreadyexists")
)

var strReg = fmt.Sprintf(`https:\/\/%s\/([\w_.-]*)\/([\w_.-]*)(?:\?commitID=)*(.*)`, strings.Replace(constants.PlatformURL, ".", "\\.", -1))

// ProjectURLRe Regex used to validate project fields /orgname/projectname[?commitID=someUUID]
var ProjectURLRe = regexp.MustCompile(strReg)

// VersionInfo is used in cases where we only care about parsing the version field. In all other cases the version is parsed via
// the Project struct
type VersionInfo struct {
	Branch  string `yaml:"branch"`
	Version string `yaml:"version"`
}

// ProjectSimple reflects a bare basic project structure
type ProjectSimple struct {
	Project string `yaml:"project"`
}

// Project covers the top level project structure of our yaml
type Project struct {
	Project      string        `yaml:"project"`
	Namespace    string        `yaml:"namespace,omitempty"`
	Branch       string        `yaml:"branch,omitempty"`
	Version      string        `yaml:"version,omitempty"`
	Environments string        `yaml:"environments,omitempty"`
	Platforms    []Platform    `yaml:"platforms,omitempty"`
	Languages    []Language    `yaml:"languages,omitempty"`
	Constants    []*Constant   `yaml:"constants,omitempty"`
	Secrets      *SecretScopes `yaml:"secrets,omitempty"`
	Events       []Event       `yaml:"events,omitempty"`
	Scripts      []Script      `yaml:"scripts,omitempty"`
	path         string        // "private"

	// Deprecated
	Variables interface{} `yaml:"variables,omitempty"`
	Owner     string      `yaml:"owner,omitempty"`
	Name      string      `yaml:"name,omitempty"`
}

// tracks deprecation warning; remove as soon as possible
var warned bool

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

// Constant covers the constant structure, which goes under Project
type Constant struct {
	Name        string     `yaml:"name"`
	Value       string     `yaml:"value"`
	Constraints Constraint `yaml:"constraints,omitempty"`
}

// SecretScopes holds secret scopes, scopes define what the secrets belong to
type SecretScopes struct {
	User    []*Secret `yaml:"user,omitempty"`
	Project []*Secret `yaml:"project,omitempty"`
}

// Secret covers the variable structure, which goes under Project
type Secret struct {
	Name        string     `yaml:"name"`
	Description string     `yaml:"description"`
	Constraints Constraint `yaml:"constraints"`
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
	Description string     `yaml:"description"`
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

	if project.Variables != nil {
		return nil, FailValidate.New("variable_field_deprecation_warning")
	}

	if project.Project == "" && project.Owner != "" && project.Name != "" {
		if !warned {
			print.Warning(locale.Tr("warn_deprecation_owner_name_fields", project.Owner, project.Name))
			warned = true
		}
		project.Project = fmt.Sprintf("https://%s/%s/%s", constants.PlatformURL, project.Owner, project.Name)
	}

	return &project, nil
}

// Path returns the project's activestate.yaml file path.
func (p *Project) Path() string {
	return p.path
}

// SetPath sets the path of the project file and should generally only be used by tests
func (p *Project) SetPath(path string) {
	p.path = path
}

// ValidateProjectURL validates the configured project URL
func ValidateProjectURL(url string) *failures.Failure {
	match := ProjectURLRe.FindStringSubmatch(url)
	if len(match) < 3 {
		return FailParseProject.New(locale.T("err_bad_project_url"))
	}
	return nil
}

// Save the project to its activestate.yaml file
func (p *Project) Save() *failures.Failure {
	dat, err := yaml.Marshal(p)
	if err != nil {
		return failures.FailMarshal.Wrap(err)
	}

	fail := ValidateProjectURL(p.Project)
	if fail != nil {
		return fail
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

// SetCommit sets the commit id within the current project file. This is done
// in-place so that line order is preserved.
func (p *Project) SetCommit(commitID string) *failures.Failure {
	fp, fail := getProjectFilePath()
	if fail != nil {
		return fail
	}

	data, err := ioutil.ReadFile(fp)
	if err != nil {
		return failures.FailOS.Wrap(err)
	}

	out, fail := setCommitInYAML(data, commitID)
	if fail != nil {
		return fail
	}

	if err := ioutil.WriteFile(fp, out, 0664); err != nil {
		return failures.FailOS.Wrap(err)
	}

	Reset()
	return nil
}

var (
	// regex captures from "project:" (at start of line) to last "/" and
	// everything after until a "?" or newline is reached. Everything after
	// that is targeted, but not captured so that only the first capture
	// group can be used in the replace value.
	setCommitRE = regexp.MustCompile(`(?m:^(project:.*\/[^?\n]*).*)`)
)

func setCommitInYAML(data []byte, commitID string) ([]byte, *failures.Failure) {
	if commitID == "" {
		return nil, failures.FailDeveloper.New("commitID must not be empty")
	}
	commitQryParam := []byte("$1?commitID=" + commitID)

	out := setCommitRE.ReplaceAll(data, commitQryParam)
	if !strings.Contains(string(out), commitID) {
		return nil, FailSetCommitID.New("err_set_commit_id")
	}

	return out, nil
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

	project, fail := GetOnce()
	if fail != nil {
		return nil, fail
	}

	project.Persist()
	return project, nil
}

// GetOnce returns the project configuration in a safe manner (returns error), the same as GetSafe, but it avoids persisting the project
func GetOnce() (*Project, *failures.Failure) {
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
		return nil, fail
	}

	return project, nil
}

// Create a new activestate.yaml with default content
func Create(projectURL string, path string) (*Project, *failures.Failure) {
	if path == "" {
		return nil, FailNewBlankPath.New(locale.T("err_project_require_path"))
	}
	path = filepath.Join(path, constants.ConfigFileName)

	if fileutils.FileExists(path) {
		return nil, FailProjectExists.New(locale.T("err_projectfile_exists"))
	}

	fail := ValidateProjectURL(projectURL)
	if fail != nil {
		return nil, fail
	}
	match := ProjectURLRe.FindStringSubmatch(projectURL)
	owner, project := match[1], match[2]

	data := map[string]interface{}{
		"Project": projectURL,
		"Content": locale.T("sample_yaml",
			map[string]interface{}{"Owner": owner, "Project": project}),
	}

	content, fail := loadTemplate(path, data)
	if fail != nil {
		return nil, fail
	}

	fail = fileutils.WriteFile(path, []byte(content.String()))
	if fail != nil {
		return nil, fail
	}

	return Parse(path)
}

// ParseVersionInfo parses the version field from the projectfile, and ONLY the version field. This is to ensure it doesn't
// trip over older activestate.yaml's with breaking changes
func ParseVersionInfo() (*VersionInfo, *failures.Failure) {
	var projectFilePath string
	if persistentProject != nil {
		projectFilePath = persistentProject.Path()
	} else {
		var fail *failures.Failure
		projectFilePath, fail = getProjectFilePath()
		if fail != nil {
			// Not being able to find a project file is not a failure for the purposes of this function
			return nil, nil
		}
	}

	if projectFilePath == "" {
		return nil, nil
	}

	dat, err := ioutil.ReadFile(projectFilePath)
	if err != nil {
		return nil, failures.FailIO.Wrap(err)
	}

	versionStruct := VersionInfo{}
	err = yaml.Unmarshal([]byte(dat), &versionStruct)
	if err != nil {
		return nil, FailParseProject.Wrap(err)
	}

	if versionStruct.Version == "" {
		return nil, nil
	}

	version := strings.TrimSpace(versionStruct.Version)
	match, fail := regexp.MatchString("^\\d+\\.\\d+\\.\\d+-\\d+$", version)
	if fail != nil || !match {
		return &versionStruct, FailInvalidVersion.New(locale.T("err_invalid_version"))
	}

	return &versionStruct, nil
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
	if p.Project == "" {
		failures.Handle(failures.FailDeveloper.New("err_persist_invalid_project"), locale.T("err_invalid_project"))
		os.Exit(1)
	}
	persistentProject = p
	os.Setenv(constants.ProjectEnvVarName, p.Path())
}
