package projectfile

import (
	"errors"
	"fmt"
	"io/ioutil"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/ActiveState/cli/internal/assets"
	"github.com/ActiveState/cli/internal/condition"
	"github.com/ActiveState/cli/internal/config"
	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/hash"
	"github.com/ActiveState/cli/internal/language"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/multilog"
	"github.com/ActiveState/cli/internal/osutils"
	"github.com/ActiveState/cli/internal/profile"
	"github.com/ActiveState/cli/internal/rollbar"
	"github.com/ActiveState/cli/internal/rtutils"
	"github.com/ActiveState/cli/internal/sliceutils"
	"github.com/ActiveState/cli/internal/strutils"
	"github.com/ActiveState/cli/pkg/sysinfo"
	"github.com/go-openapi/strfmt"
	"github.com/google/uuid"
	"github.com/imdario/mergo"
	"github.com/spf13/cast"
	"github.com/thoas/go-funk"
	"gopkg.in/yaml.v2"
)

var (
	urlProjectRegexStr = fmt.Sprintf(`https:\/\/[\w\.]+\/([\w_.-]*)\/([\w_.-]*)(?:\?commitID=)*([^&]*)(?:\&branch=)*(.*)`)
	urlCommitRegexStr  = fmt.Sprintf(`https:\/\/[\w\.]+\/commit\/(.*)`)

	// ProjectURLRe Regex used to validate project fields /orgname/projectname[?commitID=someUUID]
	ProjectURLRe = regexp.MustCompile(urlProjectRegexStr)
	// CommitURLRe Regex used to validate commit info /commit/someUUID
	CommitURLRe = regexp.MustCompile(urlCommitRegexStr)
)

type ErrorParseProject struct{ *locale.LocalizedError }

type ErrorNoProject struct{ *locale.LocalizedError }

type ErrorNoProjectFromEnv struct{ *locale.LocalizedError }

// projectURL comprises all fields of a parsed project URL
type projectURL struct {
	Owner      string
	Name       string
	CommitID   string
	BranchName string
}

var projectMapMutex = &sync.Mutex{}

const LocalProjectsConfigKey = "projects"

// VersionInfo is used in cases where we only care about parsing the version field. In all other cases the version is parsed via
// the Project struct
type VersionInfo struct {
	Branch  string
	Version string
	Lock    string `yaml:"lock"`
}

// ProjectSimple reflects a bare basic project structure
type ProjectSimple struct {
	Project string `yaml:"project"`
}

// Project covers the top level project structure of our yaml
type Project struct {
	Project       string        `yaml:"project"`
	Lock          string        `yaml:"lock,omitempty"`
	Environments  string        `yaml:"environments,omitempty"`
	Platforms     []Platform    `yaml:"platforms,omitempty"`
	Languages     Languages     `yaml:"languages,omitempty"`
	Constants     Constants     `yaml:"constants,omitempty"`
	Secrets       *SecretScopes `yaml:"secrets,omitempty"`
	Events        Events        `yaml:"events,omitempty"`
	Scripts       Scripts       `yaml:"scripts,omitempty"`
	Jobs          Jobs          `yaml:"jobs,omitempty"`
	Private       bool          `yaml:"private,omitempty"`
	path          string        // "private"
	parsedURL     projectURL    // parsed url data
	parsedBranch  string
	parsedVersion string
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
			multilog.Error("UUID generation failed, defaulting to serialization")
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
func Parse(configFilepath string) (_ *Project, rerr error) {
	projectDir := filepath.Dir(configFilepath)
	files, err := ioutil.ReadDir(projectDir)
	if err != nil {
		return nil, locale.WrapError(err, "err_project_readdir", "Could not read project directory: {{.V0}}.", projectDir)
	}

	project, err := parse(configFilepath)
	if err != nil {
		return nil, err
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
			continue
		}

		secondaryProject, err := parse(filepath.Join(projectDir, file.Name()))
		if err != nil {
			return nil, err
		}
		if err := mergo.Merge(project, *secondaryProject, mergo.WithAppendSlice, mergo.WithOverride); err != nil {
			return nil, errs.Wrap(err, "Could not merge %s into your activestate.yaml", file.Name())
		}

		// Now reverse all arrays such that any members that were redefined show up first. (The array
		// will still have two copies.)
		if list := project.Platforms; len(secondaryProject.Platforms) > 0 {
			for i, j := 0, len(list)-1; i < j; i, j = i+1, j-1 {
				list[i], list[j] = list[j], list[i]
			}
		}
		if list := project.Languages; len(secondaryProject.Languages) > 0 {
			for i, j := 0, len(list)-1; i < j; i, j = i+1, j-1 {
				list[i], list[j] = list[j], list[i]
			}
		}
		if list := project.Constants; len(secondaryProject.Constants) > 0 {
			for i, j := 0, len(list)-1; i < j; i, j = i+1, j-1 {
				list[i], list[j] = list[j], list[i]
			}
		}
		if secondaryProject.Secrets != nil {
			if list := project.Secrets.User; len(secondaryProject.Secrets.User) > 0 {
				for i, j := 0, len(list)-1; i < j; i, j = i+1, j-1 {
					list[i], list[j] = list[j], list[i]
				}
			}
			if list := project.Secrets.Project; len(secondaryProject.Secrets.Project) > 0 {
				for i, j := 0, len(list)-1; i < j; i, j = i+1, j-1 {
					list[i], list[j] = list[j], list[i]
				}
			}
		}
		if list := project.Events; len(secondaryProject.Events) > 0 {
			for i, j := 0, len(list)-1; i < j; i, j = i+1, j-1 {
				list[i], list[j] = list[j], list[i]
			}
		}
		if list := project.Jobs; len(secondaryProject.Jobs) > 0 {
			for i, j := 0, len(list)-1; i < j; i, j = i+1, j-1 {
				list[i], list[j] = list[j], list[i]
			}
		}
	}

	if err = project.Init(); err != nil {
		return nil, errs.Wrap(err, "project.Init failed")
	}

	cfg, err := config.New()
	if err != nil {
		return nil, errs.Wrap(err, "Could not read configuration required by projectfile parser.")
	}
	defer rtutils.Closer(cfg.Close, &rerr)

	namespace := fmt.Sprintf("%s/%s", project.parsedURL.Owner, project.parsedURL.Name)
	StoreProjectMapping(cfg, namespace, filepath.Dir(project.Path()))

	return project, nil
}

// Init initializes the parsedURL field from the project url string
func (p *Project) Init() error {
	parsedURL, err := p.parseURL()
	if err != nil {
		return locale.WrapInputError(err, "parse_project_file_url_err", "Could not parse project url: {{.V0}}.", p.Project)
	}
	p.parsedURL = parsedURL

	logging.Debug("Parsed URL: %v", p.parsedURL)

	// Ensure branch name is set
	if p.parsedURL.Owner != "" && p.parsedURL.BranchName == "" {
		logging.Debug("Appending default branch as none is set")
		if err := p.SetBranch(constants.DefaultBranchName); err != nil {
			return locale.WrapError(err, "err_set_default_branch", "", constants.DefaultBranchName)
		}
	}

	if p.Lock != "" {
		parsedLock, err := ParseLock(p.Lock)
		if err != nil {
			return errs.Wrap(err, "ParseLock %s failed", p.Lock)
		}

		p.parsedBranch = parsedLock.Branch
		p.parsedVersion = parsedLock.Version
	}

	return nil
}

func parse(configFilepath string) (*Project, error) {
	dat, err := ioutil.ReadFile(configFilepath)
	if err != nil {
		return nil, errs.Wrap(err, "ioutil.ReadFile %s failure", configFilepath)
	}

	project := Project{}
	err = yaml.Unmarshal([]byte(dat), &project)
	project.path = configFilepath

	if err != nil {
		return nil, &ErrorParseProject{locale.NewError(
			"err_project_parsed",
			"Project file `{{.V1}}` could not be parsed, the parser produced the following error: {{.V0}}", err.Error(), configFilepath),
		}
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

// BranchName returns the branch name specified in the project
func (p *Project) BranchName() string {
	return p.parsedURL.BranchName
}

// Path returns the project's activestate.yaml file path.
func (p *Project) Path() string {
	return p.path
}

// SetPath sets the path of the project file and should generally only be used by tests
func (p *Project) SetPath(path string) {
	p.path = path
}

// VersionBranch returns the branch as it was interpreted from the lock
func (p *Project) VersionBranch() string {
	return p.parsedBranch
}

// Version returns the branch as it was interpreted from the lock
func (p *Project) Version() string {
	return p.parsedVersion
}

// ValidateProjectURL validates the configured project URL
func ValidateProjectURL(url string) error {
	// Note: This line also matches headless commit URLs: match == {'commit', '<commit_id>'}
	match := ProjectURLRe.FindStringSubmatch(url)
	if len(match) < 3 {
		return &ErrorParseProject{locale.NewError("err_bad_project_url")}
	}
	return nil
}

// Reload the project file from disk
func (p *Project) Reload() error {
	pj, err := Parse(p.path)
	if err != nil {
		return err
	}
	*p = *pj
	return nil
}

// Save the project to its activestate.yaml file
func (p *Project) Save(cfg ConfigGetter) error {
	return p.save(cfg, p.Path())
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

func parseURL(rawURL string) (projectURL, error) {
	p := projectURL{}

	err := ValidateProjectURL(rawURL)
	if err != nil {
		return p, err
	}

	u, err := url.Parse(rawURL)
	if err != nil {
		return p, errs.Wrap(err, "Could not parse URL")
	}

	path := strings.Split(u.Path, "/")
	if len(path) > 2 {
		if path[1] == "commit" {
			p.CommitID = path[2]
		} else {
			p.Owner = path[1]
			p.Name = path[2]
		}
	}

	q := u.Query()
	if c := q.Get("commitID"); c != "" {
		p.CommitID = c
	}

	if p.CommitID != "" {
		if err := validateUUID(p.CommitID); err != nil {
			return p, err
		}
	}

	if b := q.Get("branch"); b != "" {
		p.BranchName = b
	}

	return p, nil
}

// Save the project to its activestate.yaml file
func (p *Project) save(cfg ConfigGetter, path string) error {
	dat, err := yaml.Marshal(p)
	if err != nil {
		return errs.Wrap(err, "yaml.Marshal failed")
	}

	err = ValidateProjectURL(p.Project)
	if err != nil {
		return errs.Wrap(err, "ValidateProjectURL failed")
	}

	logging.Debug("Saving %s", path)

	f, err := os.Create(path)
	if err != nil {
		return errs.Wrap(err, "os.Create %s failed", path)
	}
	defer f.Close()

	_, err = f.Write([]byte(dat))
	if err != nil {
		return errs.Wrap(err, "f.Write %s failed", path)
	}

	StoreProjectMapping(cfg, fmt.Sprintf("%s/%s", p.parsedURL.Owner, p.parsedURL.Name), filepath.Dir(p.Path()))

	return nil
}

// SetNamespace updates the namespace in the project file
func (p *Project) SetNamespace(owner, project string) error {
	pf := NewProjectField()
	if err := pf.LoadProject(p.Project); err != nil {
		return errs.Wrap(err, "Could not load activestate.yaml")
	}
	pf.SetNamespace(owner, project)
	if err := pf.Save(p.path); err != nil {
		return errs.Wrap(err, "Could not save activestate.yaml")
	}

	// keep parsed url components in sync
	p.parsedURL.Owner = owner
	p.parsedURL.Name = project
	p.Project = pf.String()

	return nil
}

// SetCommit sets the commit id within the current project file. This is done
// in-place so that line order is preserved.
// If headless is true, the project is defined by a commit-id only
func (p *Project) SetCommit(commitID string, headless bool) error {
	pf := NewProjectField()
	if err := pf.LoadProject(p.Project); err != nil {
		return errs.Wrap(err, "Could not load activestate.yaml")
	}
	pf.SetCommit(commitID, headless)
	if err := pf.Save(p.path); err != nil {
		return errs.Wrap(err, "Could not save activestate.yaml")
	}

	p.parsedURL.CommitID = commitID
	p.Project = pf.String()
	return nil
}

// SetBranch sets the branch within the current project file. This is done
// in-place so that line order is preserved.
func (p *Project) SetBranch(branch string) error {
	pf := NewProjectField()

	if err := pf.LoadProject(p.Project); err != nil {
		return errs.Wrap(err, "Could not load activestate.yaml")
	}

	pf.SetBranch(branch)

	if !condition.InUnitTest() || p.path != "" {
		if err := pf.Save(p.path); err != nil {
			return errs.Wrap(err, "Could not save activestate.yaml")
		}
	}

	p.parsedURL.BranchName = branch
	p.Project = pf.String()
	return nil
}

// GetProjectFilePath returns the path to the project activestate.yaml
func GetProjectFilePath() (string, error) {
	defer profile.Measure("GetProjectFilePath", time.Now())
	lookup := []func() (string, error){
		getProjectFilePathFromEnv,
		getProjectFilePathFromWd,
		getProjectFilePathFromDefault,
	}
	for _, getProjectFilePath := range lookup {
		path, err := getProjectFilePath()
		if err != nil {
			return "", errs.Wrap(err, "getProjectFilePath failed")
		}
		if path != "" {
			return path, nil
		}
	}

	return "", &ErrorNoProject{locale.NewInputError("err_no_projectfile")}
}

func getProjectFilePathFromEnv() (string, error) {
	projectFilePath := os.Getenv(constants.ProjectEnvVarName)
	if projectFilePath != "" {
		if fileutils.FileExists(projectFilePath) {
			return projectFilePath, nil
		}
		return "", &ErrorNoProjectFromEnv{locale.NewInputError("err_project_env_file_not_exist", "", projectFilePath)}
	}
	return "", nil
}

func getProjectFilePathFromWd() (string, error) {
	root, err := osutils.Getwd()
	if err != nil {
		return "", errs.Wrap(err, "osutils.Getwd failed")
	}

	path, err := fileutils.FindFileInPath(root, constants.ConfigFileName)
	if err != nil && !errors.Is(err, fileutils.ErrorFileNotFound) {
		return "", errs.Wrap(err, "fileutils.FindFileInPath %s failed", root)
	}

	return path, nil
}

func getProjectFilePathFromDefault() (_ string, rerr error) {
	cfg, err := config.New()
	if err != nil {
		return "", errs.Wrap(err, "Could not read configuration required to determine default project")
	}
	defer rtutils.Closer(cfg.Close, &rerr)

	defaultProjectPath := cfg.GetString(constants.GlobalDefaultPrefname)
	if defaultProjectPath == "" {
		return "", nil
	}

	path, err := fileutils.FindFileInPath(defaultProjectPath, constants.ConfigFileName)
	if err != nil && !errors.Is(err, fileutils.ErrorFileNotFound) {
		return "", errs.Wrap(err, "fileutils.FindFileInPath %s failed", defaultProjectPath)
	}
	return path, nil
}

// Get returns the project configration in an unsafe manner (exits if errors occur)
func Get() *Project {
	project, err := GetSafe()
	if err != nil {
		multilog.Error("projectfile.Get() failed with: %s", err.Error())
		fmt.Fprint(os.Stderr, locale.T("err_project_file_unavailable"))
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

	project, err := GetOnce()
	if err != nil {
		return nil, err
	}

	project.Persist()
	return project, nil
}

// GetOnce returns the project configuration in a safe manner (returns error), the same as GetSafe, but it avoids persisting the project
func GetOnce() (*Project, error) {
	// we do not want to use a path provided by state if we're running tests
	projectFilePath, err := GetProjectFilePath()
	if err != nil {
		if errors.Is(err, fileutils.ErrorFileNotFound) {
			return nil, &ErrorNoProject{locale.WrapError(err, "err_project_file_notfound", "Could not detect project file path.")}
		}
		return nil, err
	}

	project, err := Parse(projectFilePath)
	if err != nil {
		return nil, errs.Wrap(err, "Parse %s failed", projectFilePath)
	}

	return project, nil
}

// FromPath will return the projectfile that's located at the given path (this will walk up the directory tree until it finds the project)
func FromPath(path string) (*Project, error) {
	defer profile.Measure("projectfile:FromPath", time.Now())
	// we do not want to use a path provided by state if we're running tests
	projectFilePath, err := fileutils.FindFileInPath(path, constants.ConfigFileName)
	if err != nil {
		return nil, &ErrorNoProject{locale.WrapInputError(err, "err_no_projectfile")}
	}

	_, err = ioutil.ReadFile(projectFilePath)
	if err != nil {
		logging.Warning("Cannot load config file: %v", err)
		return nil, &ErrorNoProject{locale.WrapInputError(err, "err_no_projectfile")}
	}
	project, err := Parse(projectFilePath)
	if err != nil {
		return nil, errs.Wrap(err, "Parse %s failed", projectFilePath)
	}

	return project, nil
}

// FromExactPath will return the projectfile that's located at the given path without walking up the directory tree
func FromExactPath(path string) (*Project, error) {
	// we do not want to use a path provided by state if we're running tests
	projectFilePath := filepath.Join(path, constants.ConfigFileName)

	if !fileutils.FileExists(projectFilePath) {
		return nil, &ErrorNoProject{locale.NewInputError("err_no_projectfile")}
	}

	_, err := ioutil.ReadFile(projectFilePath)
	if err != nil {
		logging.Warning("Cannot load config file: %v", err)
		return nil, &ErrorNoProject{locale.WrapInputError(err, "err_no_projectfile")}
	}
	project, err := Parse(projectFilePath)
	if err != nil {
		return nil, errs.Wrap(err, "Parse %s failed", projectFilePath)
	}

	return project, nil
}

// CreateParams are parameters that we create a custom activestate.yaml file from
type CreateParams struct {
	Owner      string
	Project    string
	CommitID   *strfmt.UUID
	BranchName string
	Directory  string
	Content    string
	Language   string
	Private    bool
	path       string
	ProjectURL string
}

// TestOnlyCreateWithProjectURL a new activestate.yaml with default content
func TestOnlyCreateWithProjectURL(projectURL, path string) (*Project, error) {
	return createCustom(&CreateParams{
		ProjectURL: projectURL,
		Directory:  path,
	}, language.Python3)
}

// Create will create a new activestate.yaml with a projectURL for the given details
func Create(params *CreateParams) (*Project, error) {
	lang := language.MakeByName(params.Language)
	err := validateCreateParams(params)
	if err != nil {
		return nil, err
	}

	return createCustom(params, lang)
}

func createCustom(params *CreateParams, lang language.Language) (*Project, error) {
	err := fileutils.MkdirUnlessExists(params.Directory)
	if err != nil {
		return nil, err
	}

	var commitID string
	if params.ProjectURL == "" {
		u, err := url.Parse(fmt.Sprintf("https://%s/%s/%s", constants.PlatformURL, params.Owner, params.Project))
		if err != nil {
			return nil, errs.Wrap(err, "url parse new project url failed")
		}
		q := u.Query()

		if params.CommitID != nil {
			commitID = params.CommitID.String()
			q.Set("commitID", commitID)
		}
		if params.BranchName != "" {
			q.Set("branch", params.BranchName)
		}

		u.RawQuery = q.Encode()
		params.ProjectURL = u.String()
	}

	params.path = filepath.Join(params.Directory, constants.ConfigFileName)
	if fileutils.FileExists(params.path) {
		return nil, locale.NewInputError("err_projectfile_exists")
	}

	err = ValidateProjectURL(params.ProjectURL)
	if err != nil {
		return nil, err
	}
	match := ProjectURLRe.FindStringSubmatch(params.ProjectURL)
	if len(match) < 3 {
		return nil, locale.NewInputError("err_projectfile_invalid_url")
	}
	owner, project := match[1], match[2]

	shell := "bash"
	if runtime.GOOS == "windows" {
		shell = "batch"
	}

	content := params.Content
	if content == "" && lang != language.Unset && lang != language.Unknown {
		tplName := "activestate.yaml." + strings.TrimRight(lang.String(), "23") + ".tpl"
		template, err := assets.ReadFileBytes(tplName)
		if err != nil {
			return nil, errs.Wrap(err, "Could not read asset")
		}
		content, err = strutils.ParseTemplate(
			string(template),
			map[string]interface{}{"Owner": owner, "Project": project, "Shell": shell, "Language": lang.String(), "LangExe": lang.Executable().Filename()})
		if err != nil {
			return nil, errs.Wrap(err, "Could not parse %s", tplName)
		}
	}

	data := map[string]interface{}{
		"Project":  params.ProjectURL,
		"CommitID": commitID,
		"Content":  content,
		"Private":  params.Private,
	}

	tplName := "activestate.yaml.tpl"
	tplContents, err := assets.ReadFileBytes(tplName)
	if err != nil {
		return nil, errs.Wrap(err, "Could not read asset")
	}
	fileContents, err := strutils.ParseTemplate(string(tplContents), data)
	if err != nil {
		return nil, errs.Wrap(err, "Could not parse %s", tplName)
	}

	err = fileutils.WriteFile(params.path, []byte(fileContents))
	if err != nil {
		return nil, err
	}

	return Parse(params.path)
}

func validateCreateParams(params *CreateParams) error {
	switch {
	case params.Directory == "":
		return locale.NewInputError("err_project_require_path")
	case params.ProjectURL != "":
		return nil // Owner and Project not required when projectURL is set
	case params.Owner == "":
		return locale.NewInputError("err_project_require_owner")
	case params.Project == "":
		return locale.NewInputError("err_project_require_name")
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
		return nil, errs.Wrap(err, "ioutil.ReadFile %s failed", projectFilePath)
	}

	versionStruct := VersionInfo{}
	err = yaml.Unmarshal(dat, &versionStruct)
	if err != nil {
		return nil, &ErrorParseProject{locale.WrapError(err, "Could not unmarshal activestate.yaml")}
	}

	if versionStruct.Lock == "" {
		return nil, nil
	}

	return ParseLock(versionStruct.Lock)
}

func ParseLock(lock string) (*VersionInfo, error) {
	split := strings.Split(lock, "@")
	if len(split) != 2 {
		return nil, locale.NewInputError("err_invalid_lock", "", lock)
	}

	return &VersionInfo{
		Branch:  split[0],
		Version: split[1],
		Lock:    lock,
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
		multilog.Error("projectfile.Persist() failed because no project is defined")
		fmt.Fprint(os.Stderr, locale.T("err_invalid_project"))
		os.Exit(1)
	}
	persistentProject = p
	os.Setenv(constants.ProjectEnvVarName, p.Path())
}

type ConfigGetter interface {
	GetStringMapStringSlice(key string) map[string][]string
	AllKeys() []string
	GetStringSlice(string) []string
	Set(string, interface{}) error
	GetThenSet(string, func(interface{}) (interface{}, error)) error
	Close() error
}

func GetProjectMapping(config ConfigGetter) map[string][]string {
	addDeprecatedProjectMappings(config)
	CleanProjectMapping(config)
	projects := config.GetStringMapStringSlice(LocalProjectsConfigKey)
	if projects == nil {
		return map[string][]string{}
	}
	return projects
}

func GetProjectFileMapping(config ConfigGetter) map[string][]*Project {
	projects := GetProjectMapping(config)

	res := make(map[string][]*Project)
	for name, paths := range projects {
		if name == "/" {
			continue
		}
		var pFiles []*Project
		for _, path := range paths {
			prj, err := FromExactPath(path)
			if err != nil {
				multilog.Error("Could not read project file at %s: %v", path, err)
				continue
			}
			pFiles = append(pFiles, prj)
		}
		if len(pFiles) > 0 {
			res[name] = pFiles
		}
	}
	return res
}

func GetCachedProjectNameForPath(config ConfigGetter, projectPath string) string {
	projects := GetProjectMapping(config)

	for name, paths := range projects {
		if name == "/" {
			continue
		}
		for _, path := range paths {
			if isEqual, err := fileutils.PathsEqual(projectPath, path); isEqual {
				if err != nil {
					logging.Debug("Failed to compare paths %s and %s", projectPath, path)
				}
				return name
			}
		}
	}
	return ""
}

func addDeprecatedProjectMappings(cfg ConfigGetter) {
	var unsets []string

	err := cfg.GetThenSet(
		LocalProjectsConfigKey,
		func(v interface{}) (interface{}, error) {
			projects, err := cast.ToStringMapStringSliceE(v)
			if err != nil && v != nil { // don't report if error due to nil input
				multilog.Log(logging.ErrorNoStacktrace, rollbar.Error)("Projects data in config is abnormal (type: %T)", v)
			}

			keys := funk.FilterString(cfg.AllKeys(), func(v string) bool {
				return strings.HasPrefix(v, "project_")
			})

			for _, key := range keys {
				namespace := strings.TrimPrefix(key, "project_")
				newPaths := projects[namespace]
				paths := cfg.GetStringSlice(key)
				projects[namespace] = funk.UniqString(append(newPaths, paths...))
				unsets = append(unsets, key)
			}

			return projects, nil
		},
	)
	if err != nil {
		multilog.Error("Could not update project mapping in config, error: %v", err)
	}
	for _, unset := range unsets {
		if err := cfg.Set(unset, nil); err != nil {
			multilog.Error("Could not clear config entry for key %s, error: %v", unset, err)
		}
	}

}

// GetProjectPaths returns the paths of all projects associated with the namespace
func GetProjectPaths(cfg ConfigGetter, namespace string) []string {
	projects := GetProjectMapping(cfg)

	// match case-insensitively
	var paths []string
	for key, value := range projects {
		if strings.ToLower(key) == strings.ToLower(namespace) {
			paths = append(paths, value...)
		}
	}

	return paths
}

// StoreProjectMapping associates the namespace with the project
// path in the config
func StoreProjectMapping(cfg ConfigGetter, namespace, projectPath string) {
	err := cfg.GetThenSet(
		LocalProjectsConfigKey,
		func(v interface{}) (interface{}, error) {
			projects, err := cast.ToStringMapStringSliceE(v)
			if err != nil && v != nil { // don't report if error due to nil input
				multilog.Log(logging.ErrorNoStacktrace, rollbar.Error)("Projects data in config is abnormal (type: %T)", v)
			}

			projectPath, err = fileutils.ResolveUniquePath(projectPath)
			if err != nil {
				multilog.Log(logging.ErrorNoStacktrace, rollbar.Error)("Could not resolve uniqe project path, %v", err)
				projectPath = filepath.Clean(projectPath)
			}

			for name, paths := range projects {
				for i, path := range paths {
					path, err = fileutils.ResolveUniquePath(path)
					if err != nil {
						multilog.Log(logging.ErrorNoStacktrace, rollbar.Error)("Could not resolve unique path, :%v", err)
						path = filepath.Clean(path)
					}

					if path == projectPath {
						projects[name] = sliceutils.RemoveFromStrings(projects[name], i)
					}

					if len(projects[name]) == 0 {
						delete(projects, name)
					}
				}
			}

			paths := projects[namespace]
			if paths == nil {
				paths = make([]string, 0)
			}

			if !funk.Contains(paths, projectPath) {
				paths = append(paths, projectPath)
			}

			projects[namespace] = paths

			return projects, nil
		},
	)
	if err != nil {
		multilog.Error("Could not set project mapping in config, error: %v", err)
	}
}

// CleanProjectMapping removes projects that no longer exist
// on a user's filesystem from the projects config entry
func CleanProjectMapping(cfg ConfigGetter) {
	err := cfg.GetThenSet(
		LocalProjectsConfigKey,
		func(v interface{}) (interface{}, error) {
			projects, err := cast.ToStringMapStringSliceE(v)
			if err != nil && v != nil { // don't report if error due to nil input
				multilog.Log(logging.ErrorNoStacktrace, rollbar.Error)("Projects data in config is abnormal (type: %T)", v)
			}

			seen := make(map[string]struct{})

			for namespace, paths := range projects {
				var removals []int
				for i, path := range paths {
					if !fileutils.DirExists(path) {
						removals = append(removals, i)
					}
				}

				projects[namespace] = sliceutils.RemoveFromStrings(projects[namespace], removals...)
				if _, ok := seen[strings.ToLower(namespace)]; ok || len(projects[namespace]) == 0 {
					delete(projects, namespace)
					continue
				}
				seen[strings.ToLower(namespace)] = struct{}{}
			}

			return projects, nil
		},
	)
	if err != nil {
		logging.Debug("Could not clean project mapping in config, error: %v", err)
	}
}
