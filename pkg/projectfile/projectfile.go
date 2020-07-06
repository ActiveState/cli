package projectfile

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"gopkg.in/yaml.v2"

	"github.com/go-openapi/strfmt"

	"github.com/ActiveState/cli/internal/config"
	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/language"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/osutils"
	"github.com/ActiveState/cli/internal/print"
)

var (
	// FailNoProject identifies a failure as being due to a missing project file
	FailNoProject = failures.Type("projectfile.fail.noproject", failures.FailUser)

	// FailNoProjectFromEnv identifies a failure as being due to the project file referenced by the env not existing
	FailNoProjectFromEnv = failures.Type("projectfile.fail.noprojectfromenv", failures.FailUser)

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

	// FailNewBlankOwner identifies a failure as being caused by the owner parameter not being set
	FailNewBlankOwner = failures.Type("projectfile.fail.blanknewonwer")

	// FailNewBlankProject identifies a failure as being caused by the project parameter not being set
	FailNewBlankProject = failures.Type("projectfile.fail.blanknewproject")

	// FailProjectExists identifies a failure as being caused by the commit id not getting set
	FailProjectExists = failures.Type("projectfile.fail.projectalreadyexists")

	// FailInvalidURL identifies a failures as being caused by the project URL being invalid
	FailInvalidURL = failures.Type("projectfile.fail.invalidurl")

	// FailProjectFileRoot identifies a failure being caused by not being able to find a path to a project file
	FailProjectFileRoot = failures.Type("projectfile.fail.projectfileroot", failures.FailNonFatal)
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
	Languages    Languages     `yaml:"languages,omitempty"`
	Constants    Constants     `yaml:"constants,omitempty"`
	Secrets      *SecretScopes `yaml:"secrets,omitempty"`
	Events       Events        `yaml:"events,omitempty"`
	Scripts      Scripts       `yaml:"scripts,omitempty"`
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
	Packages    Packages   `yaml:"packages,omitempty"`
}

var _ ConstrainedEntity = Language{}

// ID returns the language name
func (l Language) ID() string {
	return l.Name
}

// ConstraintsFilter returns the language constraints
func (l Language) ConstraintsFilter() Constraint {
	return l.Constraints
}

// Languages is a slice of Language definitions
type Languages []Language

// AsConstrainedEntities boxes languages as a slice of ConstrainedEntities
func (languages Languages) AsConstrainedEntities() (items []ConstrainedEntity) {
	for i := range languages {
		items = append(items, &languages[i])
	}
	return items
}

// MakeLanguagesFromConstrainedEntities unboxes ConstraintedEntities as Languages
func MakeLanguagesFromConstrainedEntities(items []ConstrainedEntity) (languages []*Language) {
	languages = make([]*Language, 0, len(items))
	for _, v := range items {
		if o, ok := v.(*Language); ok {
			languages = append(languages, o)
		}
	}
	return languages
}

// Constant covers the constant structure, which goes under Project
type Constant struct {
	Name        string     `yaml:"name"`
	Value       string     `yaml:"value"`
	Constraints Constraint `yaml:"constraints,omitempty"`
}

var _ ConstrainedEntity = &Constant{}

// ID returns the constant name
func (c *Constant) ID() string {
	return c.Name
}

// ConstraintsFilter returns the constant constraints
func (c *Constant) ConstraintsFilter() Constraint {
	return c.Constraints
}

// Constants is a slice of constant values
type Constants []*Constant

// AsConstrainedEntities boxes constants as a slice ConstrainedEntities
func (constants Constants) AsConstrainedEntities() (items []ConstrainedEntity) {
	for _, c := range constants {
		items = append(items, c)
	}
	return items
}

// MakeConstantsFromConstrainedEntities unboxes ConstraintedEntities as Constants
func MakeConstantsFromConstrainedEntities(items []ConstrainedEntity) (constants []*Constant) {
	constants = make([]*Constant, 0, len(items))
	for _, v := range items {
		if o, ok := v.(*Constant); ok {
			constants = append(constants, o)
		}
	}
	return constants
}

// SecretScopes holds secret scopes, scopes define what the secrets belong to
type SecretScopes struct {
	User    Secrets `yaml:"user,omitempty"`
	Project Secrets `yaml:"project,omitempty"`
}

// Secret covers the variable structure, which goes under Project
type Secret struct {
	Name        string     `yaml:"name"`
	Description string     `yaml:"description"`
	Constraints Constraint `yaml:"constraints"`
}

var _ ConstrainedEntity = &Secret{}

// ID returns the secret name
func (s *Secret) ID() string {
	return s.Name
}

// ConstraintsFilter returns the secret constraints
func (s *Secret) ConstraintsFilter() Constraint {
	return s.Constraints
}

// Secrets is a slice of Secret definitions
type Secrets []*Secret

// AsConstrainedEntities box Secrets as a slice of ConstrainedEntities
func (secrets Secrets) AsConstrainedEntities() (items []ConstrainedEntity) {
	for _, s := range secrets {
		items = append(items, s)
	}
	return items
}

// MakeSecretsFromConstrainedEntities unboxes ConstraintedEntities as Secrets
func MakeSecretsFromConstrainedEntities(items []ConstrainedEntity) (secrets []*Secret) {
	secrets = make([]*Secret, 0, len(items))
	for _, v := range items {
		if o, ok := v.(*Secret); ok {
			secrets = append(secrets, o)
		}
	}
	return secrets
}

// Constraint covers the constraint structure, which can go under almost any other struct
type Constraint struct {
	OS          string `yaml:"os,omitempty"`
	Platform    string `yaml:"platform,omitempty"`
	Environment string `yaml:"environment,omitempty"`
}

// ConstrainedEntity is an entity in a project file that can be filtered with constraints
type ConstrainedEntity interface {
	// ID returns the name of the entity
	ID() string

	// ConstraintsFilter returns the specified constraints for this entity
	ConstraintsFilter() Constraint
}

// Package covers the package structure, which goes under the language struct
type Package struct {
	Name        string     `yaml:"name"`
	Version     string     `yaml:"version"`
	Constraints Constraint `yaml:"constraints,omitempty"`
	Build       Build      `yaml:"build,omitempty"`
}

var _ ConstrainedEntity = Package{}

// ID returns the package name
func (p Package) ID() string {
	return p.Name
}

// ConstraintsFilter returns the package constraints
func (p Package) ConstraintsFilter() Constraint {
	return p.Constraints
}

// Packages is a slice of Package configurations
type Packages []Package

// AsConstrainedEntities boxes Packages as a slice of ConstrainedEntities
func (packages Packages) AsConstrainedEntities() (items []ConstrainedEntity) {
	for i := range packages {
		items = append(items, &packages[i])
	}
	return items
}

// MakePackagesFromConstrainedEntities unboxes ConstraintedEntities as Packages
func MakePackagesFromConstrainedEntities(items []ConstrainedEntity) (packages []*Package) {
	packages = make([]*Package, 0, len(items))
	for _, v := range items {
		if o, ok := v.(*Package); ok {
			packages = append(packages, o)
		}
	}
	return packages
}

// Event covers the event structure, which goes under Project
type Event struct {
	Name        string     `yaml:"name"`
	Value       string     `yaml:"value"`
	Constraints Constraint `yaml:"constraints,omitempty"`
}

var _ ConstrainedEntity = Event{}

// ID returns the event name
func (e Event) ID() string {
	return e.Name
}

// ConstraintsFilter returns the event constraints
func (e Event) ConstraintsFilter() Constraint {
	return e.Constraints
}

// Events is a slice of Event definitions
type Events []Event

// AsConstrainedEntities boxes events as a slice of ConstrainedEntities
func (events Events) AsConstrainedEntities() (items []ConstrainedEntity) {
	for i := range events {
		items = append(items, &events[i])
	}
	return items
}

// MakeEventsFromConstrainedEntities unboxes ConstraintedEntities as Events
func MakeEventsFromConstrainedEntities(items []ConstrainedEntity) (events []*Event) {
	events = make([]*Event, 0, len(items))
	for _, v := range items {
		if o, ok := v.(*Event); ok {
			events = append(events, o)
		}
	}
	return events
}

// Script covers the script structure, which goes under Project
type Script struct {
	Name        string            `yaml:"name"`
	Description string            `yaml:"description,omitempty"`
	Value       string            `yaml:"value"`
	Filename    string            `yaml:"filename,omitempty"`
	Standalone  bool              `yaml:"standalone,omitempty"`
	Language    language.Language `yaml:"language,omitempty"`
	Constraints Constraint        `yaml:"constraints,omitempty"`
}

var _ ConstrainedEntity = Script{}

// ID returns the script name
func (s Script) ID() string {
	return s.Name
}

// ConstraintsFilter returns the script constraints
func (s Script) ConstraintsFilter() Constraint {
	return s.Constraints
}

// Scripts is a slice of scripts
type Scripts []Script

// AsConstrainedEntities boxes scripts as a slice of ConstrainedEntities
func (scripts Scripts) AsConstrainedEntities() (items []ConstrainedEntity) {
	for i := range scripts {
		items = append(items, &scripts[i])
	}
	return items
}

// MakeScriptsFromConstrainedEntities unboxes ConstraintedEntities as Scripts
func MakeScriptsFromConstrainedEntities(items []ConstrainedEntity) (scripts []*Script) {
	scripts = make([]*Script, 0, len(items))
	for _, v := range items {
		if o, ok := v.(*Script); ok {
			scripts = append(scripts, o)
		}
	}
	return scripts
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
		return nil, FailParseProject.New(locale.T("err_project_parse", map[string]interface{}{"Error": err.Error()}))
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

	fail := ValidateProjectURL(project.Project)
	if fail != nil {
		return nil, fail
	}

	if project.Namespace == "" {
		match := ProjectURLRe.FindStringSubmatch(project.Project)
		project.Namespace = fmt.Sprintf("%s/%s", match[1], match[2])
	}
	config.SetProject(project.Namespace, project.path)

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

// Reload the project file from disk
func (p *Project) Reload() *failures.Failure {
	pj, fail := Parse(p.path)
	if fail != nil {
		return fail
	}
	*p = *pj
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

	logging.Debug("Saving %s", p.Path())

	f, err := os.Create(p.Path())
	if err != nil {
		return failures.FailIO.Wrap(err)
	}
	defer f.Close()

	_, err = f.Write([]byte(dat))
	if err != nil {
		return failures.FailIO.Wrap(err)
	}
	config.SetProject(p.Namespace, p.path)

	return nil
}

// SetCommit sets the commit id within the current project file. This is done
// in-place so that line order is preserved.
func (p *Project) SetCommit(commitID string) *failures.Failure {
	fp, fail := GetProjectFilePath()
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
	setCommitRE = regexp.MustCompile(`(?m:^(project:.*\/[^?\r\n]*).*)`)
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

// GetProjectFilePath returns the path to the project activestate.yaml
func GetProjectFilePath() (string, *failures.Failure) {
	projectFilePath := os.Getenv(constants.ProjectEnvVarName)
	if projectFilePath != "" {
		if !fileutils.FileExists(projectFilePath) {
			return "", FailNoProjectFromEnv.New(locale.Tr("err_project_env_file_not_exist", projectFilePath))
		}
		return projectFilePath, nil
	}

	root, err := osutils.Getwd()
	if err != nil {
		logging.Warning("Could not get project root path: %v", err)
		return "", FailProjectFileRoot.Wrap(err)
	}
	path, fail := fileutils.FindFileInPath(root, constants.ConfigFileName)
	if fail != nil {
		return "", FailNoProject.Wrap(fail, locale.T("err_no_projectfile"))
	}
	return path, nil
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
	projectFilePath, fail := GetProjectFilePath()
	if fail != nil {
		if fail.Type.Matches(fileutils.FailFindInPathNotFound) {
			return nil, FailNoProject.Wrap(fail, fail.Error())
		}
		return nil, fail
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

// FromPath will return the projectfile that's located at the given path (this will walk up the directory tree until it finds the project)
func FromPath(path string) (*Project, *failures.Failure) {
	// we do not want to use a path provided by state if we're running tests
	projectFilePath, fail := fileutils.FindFileInPath(path, constants.ConfigFileName)
	if fail != nil {
		return nil, FailNoProject.Wrap(fail, locale.T("err_no_projectfile"))
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

// CreateParams are parameters that we create a custom activestate.yaml file from
type CreateParams struct {
	Owner           string
	Project         string
	CommitID        *strfmt.UUID
	Directory       string
	Content         string
	Language        string
	LanguageVersion string
	path            string
	projectURL      string
}

// CreateWithProjectURL a new activestate.yaml with default content
func CreateWithProjectURL(projectURL, path string) (*Project, *failures.Failure) {
	return createCustom(&CreateParams{
		projectURL: projectURL,
		Directory:  path,
	})
}

// Create will create a new activestate.yaml with a projectURL for the given details
func Create(params *CreateParams) *failures.Failure {
	fail := validateCreateParams(params)
	if fail != nil {
		return fail
	}

	_, fail = createCustom(params)
	if fail != nil {
		return fail
	}

	return nil
}

func createCustom(params *CreateParams) (*Project, *failures.Failure) {
	fail := fileutils.MkdirUnlessExists(params.Directory)
	if fail != nil {
		return nil, fail
	}

	if params.projectURL == "" {
		params.projectURL = fmt.Sprintf("https://%s/%s/%s", constants.PlatformURL, params.Owner, params.Project)
		if params.CommitID != nil {
			params.projectURL = fmt.Sprintf("%s?commitID=%s", params.projectURL, params.CommitID.String())
		}
	}

	params.path = filepath.Join(params.Directory, constants.ConfigFileName)
	if fileutils.FileExists(params.path) {
		return nil, FailProjectExists.New(locale.T("err_projectfile_exists"))
	}

	fail = ValidateProjectURL(params.projectURL)
	if fail != nil {
		return nil, fail
	}
	match := ProjectURLRe.FindStringSubmatch(params.projectURL)
	if len(match) < 3 {
		return nil, FailInvalidURL.New("err_projectfile_invalid_url")
	}
	owner, project := match[1], match[2]

	if params.Content == "" {
		params.Content = locale.T("sample_yaml",
			map[string]interface{}{"Owner": owner, "Project": project})
	}

	data := map[string]interface{}{
		"Project":         params.projectURL,
		"LanguageName":    params.Language,
		"LanguageVersion": params.LanguageVersion,
		"Content":         params.Content,
	}

	template, fail := loadTemplate(params.path, data)
	if fail != nil {
		return nil, fail
	}

	fail = fileutils.WriteFile(params.path, []byte(template.String()))
	if fail != nil {
		return nil, fail
	}

	return Parse(params.path)
}

func validateCreateParams(params *CreateParams) *failures.Failure {
	switch {
	case params.Owner == "":
		return FailNewBlankOwner.New("err_project_require_owner")
	case params.Project == "":
		return FailNewBlankProject.New("err_project_require_name")
	case params.Directory == "":
		return FailNewBlankPath.New(locale.T("err_project_require_path"))
	default:
		return nil
	}
}

// ParseVersionInfo parses the version field from the projectfile, and ONLY the version field. This is to ensure it doesn't
// trip over older activestate.yaml's with breaking changes
func ParseVersionInfo(projectFilePath string) (*VersionInfo, *failures.Failure) {
	if !fileutils.FileExists(projectFilePath) {
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
	match, fail := regexp.MatchString(`^\d+\.\d+\.\d+-(SHA)?[a-f0-9]+`, version)
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
