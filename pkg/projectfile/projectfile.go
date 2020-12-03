package projectfile

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"sync"

	"github.com/ActiveState/sysinfo"
	"github.com/gobuffalo/packr"
	"github.com/google/uuid"
	"github.com/imdario/mergo"
	"gopkg.in/yaml.v2"

	"github.com/go-openapi/strfmt"
	"github.com/spf13/viper"
	"github.com/thoas/go-funk"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/hash"
	"github.com/ActiveState/cli/internal/language"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/osutils"
	"github.com/ActiveState/cli/internal/sliceutils"
	"github.com/ActiveState/cli/internal/strutils"
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

var (
	urlProjectRegexStr = fmt.Sprintf(`https:\/\/[\w\.]+\/([\w_.-]*)\/([\w_.-]*)(?:\?commitID=)*(.*)`)
	urlCommitRegexStr  = fmt.Sprintf(`https:\/\/[\w\.]+\/commit\/(.*)`)

	// ProjectURLRe Regex used to validate project fields /orgname/projectname[?commitID=someUUID]
	ProjectURLRe = regexp.MustCompile(urlProjectRegexStr)
	// CommitURLRe Regex used to validate commit info /commit/someUUID
	CommitURLRe = regexp.MustCompile(urlCommitRegexStr)
)

// projectURL comprises all fields of a parsed project URL
type projectURL struct {
	Owner    string
	Name     string
	CommitID string
}

var projectMapMutex = &sync.Mutex{}

const LocalProjectsConfigKey = "projects"

// VersionInfo is used in cases where we only care about parsing the version field. In all other cases the version is parsed via
// the Project struct
type VersionInfo struct {
	Branch  string `yaml:"branch"`
	Version string `yaml:"version"`
	Lock    string `yaml:"lock"`
}

// ProjectSimple reflects a bare basic project structure
type ProjectSimple struct {
	Project string `yaml:"project"`
}

// Project covers the top level project structure of our yaml
type Project struct {
	Project      string        `yaml:"project"`
	Branch       string        `yaml:"branch,omitempty"`
	Version      string        `yaml:"version,omitempty"`
	Lock         string        `yaml:"lock,omitempty"`
	Environments string        `yaml:"environments,omitempty"`
	Platforms    []Platform    `yaml:"platforms,omitempty"`
	Languages    Languages     `yaml:"languages,omitempty"`
	Constants    Constants     `yaml:"constants,omitempty"`
	Secrets      *SecretScopes `yaml:"secrets,omitempty"`
	Events       Events        `yaml:"events,omitempty"`
	Scripts      Scripts       `yaml:"scripts,omitempty"`
	Jobs         Jobs          `yaml:"jobs,omitempty"`
	Private      bool          `yaml:"private,omitempty"`
	path         string        // "private"
	parsedURL    projectURL    // parsed url data
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
	Name        string      `yaml:"name"`
	Version     string      `yaml:"version,omitempty"`
	Conditional Conditional `yaml:"if"`
	Constraints Constraint  `yaml:"constraints,omitempty"`
	Build       Build       `yaml:"build,omitempty"`
	Packages    Packages    `yaml:"packages,omitempty"`
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

func (l Language) ConditionalFilter() Conditional {
	return l.Conditional
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
	Name        string      `yaml:"name"`
	Value       string      `yaml:"value"`
	Conditional Conditional `yaml:"if"`
	Constraints Constraint  `yaml:"constraints,omitempty"`
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

func (c *Constant) ConditionalFilter() Conditional {
	return c.Conditional
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
	Name        string      `yaml:"name"`
	Description string      `yaml:"description"`
	Conditional Conditional `yaml:"if"`
	Constraints Constraint  `yaml:"constraints"`
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

func (s *Secret) ConditionalFilter() Conditional {
	return s.Conditional
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

// Conditional is an `if` conditional that when evalutes to true enables the entity its under
// it is meant to replace Constraints
type Conditional string

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

	ConditionalFilter() Conditional
}

// Package covers the package structure, which goes under the language struct
type Package struct {
	Name        string      `yaml:"name"`
	Version     string      `yaml:"version"`
	Conditional Conditional `yaml:"if"`
	Constraints Constraint  `yaml:"constraints,omitempty"`
	Build       Build       `yaml:"build,omitempty"`
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

func (p Package) ConditionalFilter() Conditional {
	return p.Conditional
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
	Name        string      `yaml:"name"`
	Value       string      `yaml:"value"`
	Scope       []string    `yaml:"scope"`
	Conditional Conditional `yaml:"if"`
	Constraints Constraint  `yaml:"constraints,omitempty"`
	id          string
}

var _ ConstrainedEntity = Event{}

// ID returns the event name
func (e Event) ID() string {
	if e.id == "" {
		id, err := uuid.NewUUID()
		if err != nil {
			logging.Error("UUID generation failed, defaulting to serialization")
			e.id = hash.ShortHash(e.Name, e.Value, strings.Join(e.Scope, ""))
		} else {
			e.id = id.String()
		}
	}
	return e.id
}

// ConstraintsFilter returns the event constraints
func (e Event) ConstraintsFilter() Constraint {
	return e.Constraints
}

func (e Event) ConditionalFilter() Conditional {
	return e.Conditional
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
	Name        string      `yaml:"name"`
	Description string      `yaml:"description,omitempty"`
	Value       string      `yaml:"value"`
	Filename    string      `yaml:"filename,omitempty"`
	Standalone  bool        `yaml:"standalone,omitempty"`
	Language    string      `yaml:"language,omitempty"`
	Conditional Conditional `yaml:"if"`
	Constraints Constraint  `yaml:"constraints,omitempty"`
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

func (s Script) ConditionalFilter() Conditional {
	return s.Conditional
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

// Job covers the job structure, which goes under Project
type Job struct {
	Name      string   `yaml:"name"`
	Constants []string `yaml:"constants"`
	Scripts   []string `yaml:"scripts"`
}

// Jobs is a slice of jobs
type Jobs []Job

var persistentProject *Project

// Parse the given filepath, which should be the full path to an activestate.yaml file
func Parse(configFilepath string) (*Project, error) {
	projectDir := filepath.Dir(configFilepath)
	files, err := ioutil.ReadDir(projectDir)
	if err != nil {
		return nil, failures.FailIO.Wrap(err, locale.Tl("err_project_readdir", "Could not check for project files in your project directory."))
	}

	project, fail := parse(configFilepath)
	if fail != nil {
		return nil, fail
	}

	re, _ := regexp.Compile(`activestate.(\w+).yaml`)
	for _, file := range files {
		match := re.FindStringSubmatch(file.Name())
		if len(match) == 0 {
			continue
		}

		// If an OS keyword was used ensure it matches our runtime
		l := strings.ToLower
		keyword := l(match[1])
		if (keyword == l(sysinfo.Linux.String()) || keyword == l(sysinfo.Mac.String()) || keyword == l(sysinfo.Windows.String())) &&
			keyword != l(sysinfo.OS().String()) {
			logging.Debug("Not merging %s as we're not on %s", file.Name(), sysinfo.OS().String())
			continue
		}

		secondaryProject, fail := parse(filepath.Join(projectDir, file.Name()))
		if fail != nil {
			return nil, fail
		}
		if err := mergo.Merge(project, *secondaryProject, mergo.WithAppendSlice); err != nil {
			return nil, failures.FailMarshal.Wrap(err, locale.Tl("err_merge_project", "Could not merge {{.V0}} into your activestate.yaml", file.Name()))
		}
	}

	if fail = project.Init(); fail != nil {
		return nil, fail
	}

	namespace := fmt.Sprintf("%s/%s", project.parsedURL.Owner, project.parsedURL.Name)
	storeProjectMapping(namespace, filepath.Dir(project.Path()))

	return project, nil
}

// Init initializes the parsedURL field from the project url string
func (p *Project) Init() error {
	fail := ValidateProjectURL(p.Project)
	if fail != nil {
		return fail
	}

	parsedURL, err := p.parseURL()
	if err != nil {
		return failures.FailUserInput.New("parse_project_file_url_err")
	}
	p.parsedURL = parsedURL
	return nil
}

func parse(configFilepath string) (*Project, error) {
	dat, err := ioutil.ReadFile(configFilepath)
	if err != nil {
		return nil, failures.FailIO.Wrap(err)
	}

	project := Project{}
	err = yaml.Unmarshal([]byte(dat), &project)
	project.path = configFilepath

	if err != nil {
		return nil, FailParseProject.New(
			locale.Tl("err_project_parsed", "Project file `{{.V1}}` could not be parsed, the parser produced the following error: {{.V0}}", err.Error(), configFilepath))
	}

	return &project, nil
}

// Owner returns the project namespace's organization
func (p *Project) Owner() string {
	return p.parsedURL.Owner
}

// Name returns the project namespace's name
func (p *Project) Name() string {
	return p.parsedURL.Name
}

// CommitID returns the commit ID specified in the project
func (p *Project) CommitID() string {
	return p.parsedURL.CommitID
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
func ValidateProjectURL(url string) error {
	// Note: This line also matches headless commit URLs: match == {'commit', '<commit_id>'}
	match := ProjectURLRe.FindStringSubmatch(url)
	if len(match) < 3 {
		return FailParseProject.New(locale.T("err_bad_project_url"))
	}
	return nil
}

// Reload the project file from disk
func (p *Project) Reload() error {
	pj, fail := Parse(p.path)
	if fail != nil {
		return fail
	}
	*p = *pj
	return nil
}

// Save the project to its activestate.yaml file
func (p *Project) Save() error {
	return p.save(p.Path())
}

// parseURL returns the parsed fields of a Project URL
func (p *Project) parseURL() (projectURL, error) {
	return parseURL(p.Project)
}

func validateUUID(uuidStr string) error {
	if ok := strfmt.Default.Validates("uuid", uuidStr); !ok {
		return locale.NewError("invalid_uuid_val", "Invalid commit ID {{.V0}} in activestate.yaml.  You could replace it with 'latest'", uuidStr)
	}

	var uuid strfmt.UUID
	if err := uuid.UnmarshalText([]byte(uuidStr)); err != nil {
		return locale.WrapError(err, "err_commit_id_unmarshal", "Failed to unmarshal the commit id {{.V0}} read from activestate.yaml.", uuidStr)
	}

	return nil
}

func parseURL(url string) (projectURL, error) {
	fail := ValidateProjectURL(url)
	if fail != nil {
		return projectURL{}, fail.ToError()
	}

	match := CommitURLRe.FindStringSubmatch(url)
	if len(match) > 1 {
		parts := projectURL{"", "", match[1]}
		return parts, nil
	}

	match = ProjectURLRe.FindStringSubmatch(url)
	parts := projectURL{match[1], match[2], ""}
	if len(match) == 4 {
		parts.CommitID = match[3]
	}

	if parts.CommitID != "" {
		if err := validateUUID(parts.CommitID); err != nil {
			return projectURL{}, err
		}
	}

	return parts, nil
}

func removeTemporaryLanguage(data []byte) ([]byte, error) {
	languageLine := regexp.MustCompile("(?m)^languages:")
	firstNonIndentedLine := regexp.MustCompile("(?m)^[^ \t]")

	startLoc := languageLine.FindIndex(data)
	if startLoc == nil {
		return data, nil
	}
	endLoc := firstNonIndentedLine.FindIndex(data[startLoc[1]:])
	if endLoc == nil {
		return data[:startLoc[0]], nil
	}

	end := startLoc[1] + endLoc[0]
	return append(data[:startLoc[0]], data[end:]...), nil
}

// RemoveTemporaryLanguage removes the temporary language field from the as.yaml file during state push
func (p *Project) RemoveTemporaryLanguage() error {
	fp, fail := GetProjectFilePath()
	if fail != nil {
		return errs.Wrap(fail.ToError(), "Could not find the project file location.")
	}

	data, err := ioutil.ReadFile(fp)
	if err != nil {
		return errs.Wrap(err, "Failed to read project file.")
	}

	out, err := removeTemporaryLanguage(data)
	if err != nil {
		return errs.Wrap(err, "Failed to remove language field from project file.")
	}

	if err := ioutil.WriteFile(fp, out, 0664); err != nil {
		return errs.Wrap(err, "Failed to write update project file.")
	}

	fail = p.Reload()
	if fail != nil {
		return errs.Wrap(fail.ToError(), "Failed to reload project file.")
	}
	return nil
}

// Save the project to its activestate.yaml file
func (p *Project) save(path string) error {
	dat, err := yaml.Marshal(p)
	if err != nil {
		return failures.FailMarshal.Wrap(err)
	}

	fail := ValidateProjectURL(p.Project)
	if fail != nil {
		return fail
	}

	logging.Debug("Saving %s", path)

	f, err := os.Create(path)
	if err != nil {
		return failures.FailIO.Wrap(err)
	}
	defer f.Close()

	_, err = f.Write([]byte(dat))
	if err != nil {
		return failures.FailIO.Wrap(err)
	}

	storeProjectMapping(fmt.Sprintf("%s/%s", p.parsedURL.Owner, p.parsedURL.Name), filepath.Dir(p.Path()))

	return nil
}

// SetNamespace updates the namespace in the project file
func (p *Project) SetNamespace(owner, project string) error {
	data, err := ioutil.ReadFile(p.path)
	if err != nil {
		return errs.Wrap(err, "Failed to read project file %s.", p.path)
	}

	namespace := fmt.Sprintf("%s/%s", owner, project)
	out, err := setNamespaceInYAML(data, namespace, p.CommitID())
	if err != nil {
		return errs.Wrap(err, "Failed to update namespace in project file.")
	}

	// keep parsed url components in sync
	p.parsedURL.Owner = owner
	p.parsedURL.Name = project

	if err := ioutil.WriteFile(p.path, out, 0664); err != nil {
		return errs.Wrap(err, "Failed to write project file %s", p.path)
	}

	fail := p.Reload()
	if fail != nil {
		return errs.Wrap(fail.ToError(), "Failed to reload updated projectfile.")
	}
	return nil
}

// SetCommit sets the commit id within the current project file. This is done
// in-place so that line order is preserved.
// If headless is true, the project is defined by a commit-id only
func (p *Project) SetCommit(commitID string, headless bool) error {
	data, err := ioutil.ReadFile(p.path)
	if err != nil {
		return failures.FailOS.Wrap(err)
	}

	out, fail := setCommitInYAML(data, commitID, headless)
	if fail != nil {
		return fail
	}
	p.parsedURL.CommitID = commitID

	if err := ioutil.WriteFile(p.path, out, 0664); err != nil {
		return failures.FailOS.Wrap(err)
	}

	return p.Reload()
}

var (
	// regex captures three groups:
	// 1. from "project:" (at start of line) to protocol ("https://")
	// 2. the domain name
	// 3. the url part until a "?" or newline is reached.
	// Everything after that is targeted, but not captured so that only the first three capture
	// groups can be used in the replace value.
	setCommitRE = regexp.MustCompile(`(?m:^(project: *https?:\/\/)([^\/]*\/)(.*\/[^?\r\n]*).*)`)
)

func setNamespaceInYAML(data []byte, namespace string, commitID string) ([]byte, error) {
	queryParams := ""
	if commitID != "" {
		queryParams = fmt.Sprintf("?commitID=%s", commitID)
	}
	commitQryParam := []byte(fmt.Sprintf("${1}${2}%s%s", namespace, queryParams))

	out := setCommitRE.ReplaceAll(data, commitQryParam)
	if !strings.Contains(string(out), namespace) {
		return nil, locale.NewError(
			"err_set_namespace", "Failed to set namespace {{.V0}} in activestate.yaml.", namespace)
	}
	return out, nil
}

func setCommitInYAML(data []byte, commitID string, anonymous bool) ([]byte, error) {
	if commitID == "" {
		return nil, failures.FailDeveloper.New("commitID must not be empty")
	}
	commitQryParam := []byte("$1$2$3?commitID=" + commitID)
	if anonymous {
		commitQryParam = []byte(fmt.Sprintf("${1}${2}commit/%s", commitID))
	}

	out := setCommitRE.ReplaceAll(data, commitQryParam)
	if !strings.Contains(string(out), commitID) {
		return nil, FailSetCommitID.New("err_set_commit_id")
	}

	return out, nil
}

// GetProjectFilePath returns the path to the project activestate.yaml
func GetProjectFilePath() (string, error) {
	lookup := []func() (string, error){
		getProjectFilePathFromEnv,
		getProjectFilePathFromWd,
		getProjectFilePathFromDefault,
	}
	for _, getProjectFilePath := range lookup {
		path, fail := getProjectFilePath()
		if fail != nil {
			return "", fail
		}
		if path != "" {
			return path, nil
		}
	}

	return "", FailNoProject.New(locale.T("err_no_projectfile"))
}

func getProjectFilePathFromEnv() (string, error) {
	projectFilePath := os.Getenv(constants.ProjectEnvVarName)
	if projectFilePath != "" {
		if fileutils.FileExists(projectFilePath) {
			return projectFilePath, nil
		}
		return "", FailNoProjectFromEnv.New(locale.Tr("err_project_env_file_not_exist", projectFilePath))
	}
	return "", nil
}

func getProjectFilePathFromWd() (string, error) {
	root, err := osutils.Getwd()
	if err != nil {
		return "", failures.FailIO.Wrap(err, locale.Tl("err_wd", "Could not get working directory"))
	}

	path, fail := fileutils.FindFileInPath(root, constants.ConfigFileName)
	if fail != nil && !fail.Type.Matches(fileutils.FailFindInPathNotFound) {
		return "", fail
	}

	return path, nil
}

func getProjectFilePathFromDefault() (string, error) {
	defaultProjectPath := viper.GetString(constants.GlobalDefaultPrefname)
	if defaultProjectPath == "" {
		return "", nil
	}

	path, fail := fileutils.FindFileInPath(defaultProjectPath, constants.ConfigFileName)
	if fail != nil && !fail.Type.Matches(fileutils.FailFindInPathNotFound) {
		return "", fail
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

// GetPersisted gets the persisted project, if any
func GetPersisted() *Project {
	return persistentProject
}

// GetSafe returns the project configuration in a safe manner (returns error)
func GetSafe() (*Project, error) {
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
func GetOnce() (*Project, error) {
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
func FromPath(path string) (*Project, error) {
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
	Private         bool
	path            string
	projectURL      string
}

// TestOnlyCreateWithProjectURL a new activestate.yaml with default content
func TestOnlyCreateWithProjectURL(projectURL, path string) (*Project, error) {
	return createCustom(&CreateParams{
		projectURL: projectURL,
		Directory:  path,
	}, language.Python3)
}

// Create will create a new activestate.yaml with a projectURL for the given details
func Create(params *CreateParams) error {
	lang := language.MakeByName(params.Language)
	fail := validateCreateParams(params, lang)
	if fail != nil {
		return fail
	}

	_, fail = createCustom(params, lang)
	if fail != nil {
		return fail
	}

	return nil
}

func createCustom(params *CreateParams, lang language.Language) (*Project, error) {
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

	shell := "bash"
	if runtime.GOOS == "windows" {
		shell = "batch"
	}

	box := packr.NewBox("../../assets/")

	content := params.Content
	if content == "" {
		var err error
		content, err = strutils.ParseTemplate(
			box.String("activestate.yaml."+strings.TrimRight(lang.String(), "23")+".tpl"),
			map[string]interface{}{"Owner": owner, "Project": project, "Shell": shell, "Language": lang.String(), "LangExe": lang.Executable().Filename()})
		if err != nil {
			return nil, failures.FailMisc.Wrap(err)
		}
	}

	data := map[string]interface{}{
		"Project":         params.projectURL,
		"LanguageName":    params.Language,
		"LanguageVersion": params.LanguageVersion,
		"Content":         content,
		"Private":         params.Private,
	}

	fileContents, err := strutils.ParseTemplate(box.String("activestate.yaml.tpl"), data)
	if err != nil {
		return nil, failures.FailMisc.Wrap(err)
	}

	fail = fileutils.WriteFile(params.path, []byte(fileContents))
	if fail != nil {
		return nil, fail
	}

	return Parse(params.path)
}

func validateCreateParams(params *CreateParams, lang language.Language) error {
	langValidErr := lang.Validate()
	switch {
	case langValidErr != nil:
		return failures.FailUserInput.Wrap(langValidErr)
	case params.Directory == "":
		return FailNewBlankPath.New(locale.T("err_project_require_path"))
	case params.projectURL != "":
		return nil // Owner and Project not required when projectURL is set
	case params.Owner == "":
		return FailNewBlankOwner.New("err_project_require_owner")
	case params.Project == "":
		return FailNewBlankProject.New("err_project_require_name")
	default:
		return nil
	}
}

// ParseVersionInfo parses the lock field from the projectfile and updates
// the activestate.yaml if an older version representation is present
func ParseVersionInfo(projectFilePath string) (*VersionInfo, error) {
	if !fileutils.FileExists(projectFilePath) {
		return nil, nil
	}

	dat, err := ioutil.ReadFile(projectFilePath)
	if err != nil {
		return nil, failures.FailIO.Wrap(err)
	}

	versionStruct := VersionInfo{}
	err = yaml.Unmarshal(dat, &versionStruct)
	if err != nil {
		return nil, FailParseProject.Wrap(err)
	}

	if versionStruct.Branch != "" && versionStruct.Version != "" {
		err = AddLockInfo(projectFilePath, versionStruct.Branch, versionStruct.Version)
		if err != nil {
			return nil, FailParseProject.Wrap(err, locale.T("err_update_version"))
		}
		return ParseVersionInfo(projectFilePath)
	}

	if versionStruct.Lock == "" {
		return nil, nil
	}

	lock := strings.TrimSpace(versionStruct.Lock)
	match, fail := regexp.MatchString(`^([\w\/\-\.]+@)\d+\.\d+\.\d+-(SHA)?[a-f0-9]+`, lock)
	if fail != nil || !match {
		return nil, FailInvalidVersion.New(locale.T("err_invalid_version"))
	}

	split := strings.Split(versionStruct.Lock, "@")
	if len(split) != 2 {
		return nil, FailInvalidVersion.New(locale.T("err_invalid_version"))
	}

	return &VersionInfo{
		Branch:  split[0],
		Version: split[1],
	}, nil
}

// AddLockInfo adds the lock field to activestate.yaml
func AddLockInfo(projectFilePath, branch, version string) error {
	data, err := cleanVersionInfo(projectFilePath)
	if err != nil {
		return locale.WrapError(err, "err_clean_projectfile", "Could not remove old version information from projectfile", projectFilePath)
	}

	lockRegex := regexp.MustCompile(`(?m)^lock:.*`)
	if lockRegex.Match(data) {
		versionUpdate := []byte(fmt.Sprintf("lock: %s@%s", branch, version))
		replaced := lockRegex.ReplaceAll(data, versionUpdate)
		return ioutil.WriteFile(projectFilePath, replaced, 0644)
	}

	projectRegex := regexp.MustCompile(fmt.Sprintf("(?m:(^project:\\s*%s))", ProjectURLRe))
	lockString := fmt.Sprintf("%s@%s", branch, version)
	lockUpdate := []byte(fmt.Sprintf("${1}\nlock: %s", lockString))

	data, err = ioutil.ReadFile(projectFilePath)
	if err != nil {
		return err
	}

	updated := projectRegex.ReplaceAll(data, lockUpdate)

	return ioutil.WriteFile(projectFilePath, updated, 0644)
}

func cleanVersionInfo(projectFilePath string) ([]byte, error) {
	data, err := ioutil.ReadFile(projectFilePath)
	if err != nil {
		return nil, locale.WrapError(err, "err_read_projectfile", "Failed to read the activestate.yaml at: %s", projectFilePath)
	}

	branchRegex := regexp.MustCompile(`(?m:^branch:\s*\w+\n)`)
	clean := branchRegex.ReplaceAll(data, []byte(""))

	versionRegex := regexp.MustCompile(`(?m:^version:\s*\d+.\d+.\d+-[A-Za-z0-9]+\n)`)
	clean = versionRegex.ReplaceAll(clean, []byte(""))

	err = ioutil.WriteFile(projectFilePath, clean, 0644)
	if err != nil {
		return nil, locale.WrapError(err, "err_write_clean_projectfile", "Could not write cleaned projectfile information")
	}

	return clean, nil
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

type configGetter interface {
	GetStringMapStringSlice(key string) map[string][]string
}

func GetProjectNameForPath(config configGetter, projectPath string) string {
	projects := config.GetStringMapStringSlice(LocalProjectsConfigKey)
	if projects == nil {
		projects = make(map[string][]string)
	}

	for name, paths := range projects {
		if name == "/" {
			continue
		}
		for _, path := range paths {
			if isEqual, fail := fileutils.PathsEqual(projectPath, path); isEqual {
				if fail != nil {
					logging.Debug("Failed to compare paths %s and %s", projectPath, path)
				}
				return name
			}
		}
	}
	return ""
}

// storeProjectMapping associates the namespace with the project
// path in the config
func storeProjectMapping(namespace, projectPath string) {
	projectMapMutex.Lock()
	defer projectMapMutex.Unlock()

	projectPath = filepath.Clean(projectPath)

	projects := viper.GetStringMapStringSlice(LocalProjectsConfigKey)
	if projects == nil {
		projects = make(map[string][]string)
	}

	paths := projects[namespace]
	if paths == nil {
		paths = make([]string, 0)
	}

	if !funk.Contains(paths, projectPath) {
		paths = append(paths, projectPath)
	}

	projects[namespace] = paths
	viper.Set(LocalProjectsConfigKey, projects)
}

// CleanProjectMapping removes projects that no longer exist
// on a user's filesystem from the projects config entry
func CleanProjectMapping() {
	projects := viper.GetStringMapStringSlice(LocalProjectsConfigKey)
	seen := map[string]bool{}

	for namespace, paths := range projects {
		for i, path := range paths {
			if !fileutils.DirExists(path) {
				projects[namespace] = sliceutils.RemoveFromStrings(projects[namespace], i)
			}
		}
		if ok, _ := seen[strings.ToLower(namespace)]; ok || len(projects[namespace]) == 0 {
			delete(projects, namespace)
			continue
		}
		seen[strings.ToLower(namespace)] = true
	}

	viper.Set("projects", projects)
}
