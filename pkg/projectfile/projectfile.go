package projectfile

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"os/user"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
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
	"github.com/google/uuid"
	"github.com/imdario/mergo"
	"github.com/spf13/cast"
	"github.com/thoas/go-funk"
	"gopkg.in/yaml.v2"
)

var (
	urlProjectRegexStr = `https:\/\/[\w\.]+\/([\w_.-]*)\/([\w_.-]*)(?:\?commitID=)*([^&]*)(?:\&branch=)*(.*)`
	urlCommitRegexStr  = `https:\/\/[\w\.]+\/commit\/(.*)`

	// ProjectURLRe Regex used to validate project fields /orgname/projectname[?commitID=someUUID]
	ProjectURLRe = regexp.MustCompile(urlProjectRegexStr)
	// CommitURLRe Regex used to validate commit info /commit/someUUID
	CommitURLRe = regexp.MustCompile(urlCommitRegexStr)
	// deprecatedRegex covers the deprecated fields in the project file
	deprecatedRegex = regexp.MustCompile(`(?m)^\s*(?:constraints|platforms|languages):`)
	// nonAlphanumericRegex covers all non alphanumeric characters
	nonAlphanumericRegex = regexp.MustCompile(`[^a-zA-Z0-9 ]+`)
)

const ConfigVersion = 1

type MigratorFunc func(project *Project, configVersion int) (int, error)

var migrationRunning bool

var migrator MigratorFunc

func RegisterMigrator(m MigratorFunc) {
	migrator = m
}

type ErrorParseProject struct{ *locale.LocalizedError }

type ErrorNoProject struct{ *locale.LocalizedError }

type ErrorNoProjectFromEnv struct{ *locale.LocalizedError }

type ErrorNoDefaultProject struct{ *locale.LocalizedError }

// projectURL comprises all fields of a parsed project URL
type projectURL struct {
	Owner          string
	Name           string
	LegacyCommitID string
	BranchName     string
}

const LocalProjectsConfigKey = "projects"

// VersionInfo is used in cases where we only care about parsing the version and channel fields.
// In all other cases the version is parsed via the Project struct
type VersionInfo struct {
	Channel string `yaml:"branch"` // branch for backward compatibility
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
	ConfigVersion int           `yaml:"config_version"`
	Lock          string        `yaml:"lock,omitempty"`
	Environments  string        `yaml:"environments,omitempty"`
	Constants     Constants     `yaml:"constants,omitempty"`
	Secrets       *SecretScopes `yaml:"secrets,omitempty"`
	Events        Events        `yaml:"events,omitempty"`
	Scripts       Scripts       `yaml:"scripts,omitempty"`
	Jobs          Jobs          `yaml:"jobs,omitempty"`
	Private       bool          `yaml:"private,omitempty"`
	Cache         string        `yaml:"cache,omitempty"`
	Portable      bool          `yaml:"portable,omitempty"`
	path          string        // "private"
	parsedURL     projectURL    // parsed url data
	parsedChannel string
	parsedVersion string
}

// Build covers the build map, which can go under languages or packages
// Build can hold variable keys, so we cannot predict what they are, hence why it is a map
type Build map[string]string

// ConstantFields are the common fields for the Constant type. This is required
// for type composition related to its yaml.Unmarshaler implementation.
type ConstantFields struct {
	Conditional Conditional `yaml:"if,omitempty"`
}

// Constant covers the constant structure, which goes under Project
type Constant struct {
	NameVal        `yaml:",inline"`
	ConstantFields `yaml:",inline"`
}

func (c *Constant) UnmarshalYAML(unmarshal func(interface{}) error) error {
	if err := unmarshal(&c.NameVal); err != nil {
		return err
	}
	if err := unmarshal(&c.ConstantFields); err != nil {
		return err
	}
	return nil
}

var _ ConstrainedEntity = &Constant{}

// ID returns the constant name
func (c *Constant) ID() string {
	return c.Name
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
	Conditional Conditional `yaml:"if,omitempty"`
}

var _ ConstrainedEntity = &Secret{}

// ID returns the secret name
func (s *Secret) ID() string {
	return s.Name
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

// ConstrainedEntity is an entity in a project file that can be filtered with constraints
type ConstrainedEntity interface {
	// ID returns the name of the entity
	ID() string

	ConditionalFilter() Conditional
}

// Package covers the package structure, which goes under the language struct
type Package struct {
	Name        string      `yaml:"name"`
	Version     string      `yaml:"version"`
	Conditional Conditional `yaml:"if,omitempty"`
	Build       Build       `yaml:"build,omitempty"`
}

var _ ConstrainedEntity = Package{}

// ID returns the package name
func (p Package) ID() string {
	return p.Name
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

// EventFields are the common fields for the Event type. This is required
// for type composition related to its yaml.Unmarshaler implementation.
type EventFields struct {
	Scope       []string    `yaml:"scope"`
	Conditional Conditional `yaml:"if,omitempty"`
	id          string
}

// Event covers the event structure, which goes under Project
type Event struct {
	NameVal     `yaml:",inline"`
	EventFields `yaml:",inline"`
}

func (e *Event) UnmarshalYAML(unmarshal func(interface{}) error) error {
	if err := unmarshal(&e.NameVal); err != nil {
		return err
	}
	if err := unmarshal(&e.EventFields); err != nil {
		return err
	}
	return nil
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

// ScriptFields are the common fields for the Script type. This is required
// for type composition related to its yaml.Unmarshaler implementation.
type ScriptFields struct {
	Description string      `yaml:"description,omitempty"`
	Filename    string      `yaml:"filename,omitempty"`
	Standalone  bool        `yaml:"standalone,omitempty"`
	Language    string      `yaml:"language,omitempty"`
	Conditional Conditional `yaml:"if,omitempty"`
}

// Script covers the script structure, which goes under Project
type Script struct {
	NameVal      `yaml:",inline"`
	ScriptFields `yaml:",inline"`
}

func (s *Script) UnmarshalYAML(unmarshal func(interface{}) error) error {
	if err := unmarshal(&s.NameVal); err != nil {
		return err
	}
	if err := unmarshal(&s.ScriptFields); err != nil {
		return err
	}
	return nil
}

var _ ConstrainedEntity = Script{}

// ID returns the script name
func (s Script) ID() string {
	return s.Name
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

// Parse the given filepath, which should be the full path to an activestate.yaml file
func Parse(configFilepath string) (_ *Project, rerr error) {
	projectDir := filepath.Dir(configFilepath)
	files, err := os.ReadDir(projectDir)
	if err != nil {
		return nil, locale.WrapError(err, "err_project_readdir", "Could not read project directory: {{.V0}}.", projectDir)
	}

	project, err := parse(configFilepath)
	if err != nil {
		return nil, err
	}

	re, _ := regexp.Compile(`activestate[._-](\w+)\.yaml`)
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
		if err := mergo.Merge(secondaryProject, *project, mergo.WithAppendSlice); err != nil {
			return nil, errs.Wrap(err, "Could not merge %s into your activestate.yaml", file.Name())
		}
		secondaryProject.path = project.path // keep original project path, not secondary path
		project = secondaryProject
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

	// Migrate project file if needed
	if !migrationRunning && project.ConfigVersion != ConfigVersion && migrator != nil {
		// Migrations may themselves utilize the projectfile package, so we have to ensure we don't start an infinite loop
		migrationRunning = true
		defer func() { migrationRunning = false }()

		if project.ConfigVersion > ConfigVersion {
			return nil, locale.NewInputError("err_projectfile_version_too_high")
		}
		updatedConfigVersion, errMigrate := migrator(project, ConfigVersion)

		// Ensure we update the config version regardless of any error that occurred, because we don't want to repeat
		// the same version migrations
		project.ConfigVersion = updatedConfigVersion
		if err := NewYamlField("config_version", ConfigVersion).Save(project.Path()); err != nil {
			return nil, errs.Pack(errMigrate, errs.Wrap(err, "Could not save config_version"))
		}

		if errMigrate != nil {
			return nil, errs.Wrap(errMigrate, "Migrator failed")
		}
	}

	return project, nil
}

// Init initializes the parsedURL field from the project url string
func (p *Project) Init() error {
	parsedURL, err := p.parseURL()
	if err != nil {
		return locale.WrapInputError(err, "parse_project_file_url_err", "Could not parse project url: {{.V0}}.", p.Project)
	}
	p.parsedURL = parsedURL

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

		p.parsedChannel = parsedLock.Channel
		p.parsedVersion = parsedLock.Version
	}

	return nil
}

func parse(configFilepath string) (*Project, error) {
	if !fileutils.FileExists(configFilepath) {
		return nil, &ErrorNoProject{locale.NewInputError("err_no_projectfile")}
	}

	dat, err := os.ReadFile(configFilepath)
	if err != nil {
		return nil, errs.Wrap(err, "os.ReadFile %s failure", configFilepath)
	}

	return parseData(dat, configFilepath)
}

func parseData(dat []byte, configFilepath string) (*Project, error) {
	if err := detectDeprecations(dat, configFilepath); err != nil {
		return nil, errs.Wrap(err, "deprecations found")
	}

	project := Project{}
	err2 := yaml.Unmarshal(dat, &project)
	project.path = configFilepath

	if err2 != nil {
		return nil, &ErrorParseProject{locale.NewExternalError(
			"err_project_parsed",
			"Project file `{{.V1}}` could not be parsed. The parser produced the following error: {{.V0}}", err2.Error(), configFilepath),
		}
	}

	return &project, nil
}

func detectDeprecations(dat []byte, configFilepath string) error {
	deprecations := deprecatedRegex.FindAllIndex(dat, -1)
	if len(deprecations) == 0 {
		return nil
	}
	deplist := []string{}
	for _, depIdxs := range deprecations {
		dep := strings.TrimSpace(strings.TrimSuffix(string(dat[depIdxs[0]:depIdxs[1]]), ":"))
		deplist = append(deplist, locale.Tr("pjfile_deprecation_entry", dep, strconv.Itoa(depIdxs[0])))
	}
	return &ErrorParseProject{locale.NewExternalError(
		"pjfile_deprecation_msg",
		"", configFilepath, strings.Join(deplist, "\n"), constants.DocumentationURL+"config/#deprecation"),
	}
}

// URL returns the project namespace's string URL from activestate.yaml.
func (p *Project) URL() string {
	return p.Project
}

// Owner returns the project namespace's organization
func (p *Project) Owner() string {
	return p.parsedURL.Owner
}

// Name returns the project namespace's name
func (p *Project) Name() string {
	return p.parsedURL.Name
}

// BranchName returns the branch name specified in the project
func (p *Project) BranchName() string {
	return p.parsedURL.BranchName
}

// Path returns the project's activestate.yaml file path.
func (p *Project) Path() string {
	return p.path
}

// LegacyCommitID is for use by legacy mechanics ONLY
// It returns a pre-migrated project's commit ID from activestate.yaml.
func (p *Project) LegacyCommitID() string {
	return p.parsedURL.LegacyCommitID
}

// SetLegacyCommit sets the commit id within the current project file. This is done
// in-place so that line order is preserved.
func (p *Project) SetLegacyCommit(commitID string) error {
	pf := NewProjectField()
	if err := pf.LoadProject(p.Project); err != nil {
		return errs.Wrap(err, "Could not load activestate.yaml")
	}
	pf.SetLegacyCommitID(commitID)
	if err := pf.Save(p.path); err != nil {
		return errs.Wrap(err, "Could not save activestate.yaml")
	}

	p.parsedURL.LegacyCommitID = commitID
	p.Project = pf.String()
	return nil
}

func (p *Project) Dir() string {
	return filepath.Dir(p.path)
}

// SetPath sets the path of the project file and should generally only be used by tests
func (p *Project) SetPath(path string) {
	p.path = path
}

// Channel returns the channel as it was interpreted from the lock
func (p *Project) Channel() string {
	return p.parsedChannel
}

// Version returns the version as it was interpreted from the lock
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
			p.LegacyCommitID = path[2]
		} else {
			p.Owner = path[1]
			p.Name = path[2]
		}
	}

	q := u.Query()
	if c := q.Get("commitID"); c != "" {
		p.LegacyCommitID = c
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

	if cfg != nil {
		StoreProjectMapping(cfg, fmt.Sprintf("%s/%s", p.parsedURL.Owner, p.parsedURL.Name), filepath.Dir(p.Path()))
	}

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
// It considers projects in the following order:
// 1. Environment variable (e.g. `state shell` sets one)
// 2. Working directory (i.e. walk up directory tree looking for activestate.yaml)
// 3. Fall back on default project
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
	var projectFilePath string

	if activatedProjectDirPath := os.Getenv(constants.ActivatedStateEnvVarName); activatedProjectDirPath != "" {
		projectFilePath = filepath.Join(activatedProjectDirPath, constants.ConfigFileName)
	}

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
		return "", errs.Wrap(err, "Could not read configuration required to determine which project to use")
	}
	defer rtutils.Closer(cfg.Close, &rerr)

	defaultProjectPath := cfg.GetString(constants.GlobalDefaultPrefname)
	if defaultProjectPath == "" {
		return "", nil
	}

	path, err := fileutils.FindFileInPath(defaultProjectPath, constants.ConfigFileName)
	if err != nil {
		if !errors.Is(err, fileutils.ErrorFileNotFound) {
			return "", errs.Wrap(err, "fileutils.FindFileInPath %s failed", defaultProjectPath)
		}
		return "", &ErrorNoDefaultProject{locale.NewInputError("err_no_default_project", "Could not find your project at: [ACTIONABLE]{{.V0}}[/RESET]", defaultProjectPath)}
	}
	return path, nil
}

// FromEnv returns the project configuration based on environment information (env vars, cwd, etc)
func FromEnv() (*Project, error) {
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
		return nil, errs.Wrap(err, "Could not parse projectfile")
	}

	return project, nil
}

// FromPath will return the projectfile that's located at the given path (this will walk up the directory tree until it finds the project)
func FromPath(path string) (*Project, error) {
	defer profile.Measure("projectfile:FromPath", time.Now())
	// we do not want to use a path provided by state if we're running tests
	projectFilePath, err := fileutils.FindFileInPath(path, constants.ConfigFileName)
	if err != nil {
		return nil, &ErrorNoProject{locale.WrapInputError(err, "err_project_not_found", "", path)}
	}

	_, err = os.ReadFile(projectFilePath)
	if err != nil {
		logging.Warning("Cannot load config file: %v", err)
		return nil, &ErrorNoProject{locale.WrapInputError(err, "err_no_projectfile")}
	}
	project, err := Parse(projectFilePath)
	if err != nil {
		return nil, errs.Wrap(err, "Could not parse projectfile")
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

	_, err := os.ReadFile(projectFilePath)
	if err != nil {
		logging.Warning("Cannot load config file: %v", err)
		return nil, &ErrorNoProject{locale.WrapInputError(err, "err_no_projectfile")}
	}
	project, err := Parse(projectFilePath)
	if err != nil {
		return nil, errs.Wrap(err, "Could not parse projectfile")
	}

	return project, nil
}

// CreateParams are parameters that we create a custom activestate.yaml file from
type CreateParams struct {
	Owner      string
	Project    string
	BranchName string
	Directory  string
	Content    string
	Language   string
	Private    bool
	path       string
	ProjectURL string
	Cache      string
	Portable   bool
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

	if params.ProjectURL == "" {
		// Note: cannot use api.GetPlatformURL() due to import cycle.
		host := constants.DefaultAPIHost
		if hostOverride := os.Getenv(constants.APIHostEnvVarName); hostOverride != "" {
			host = hostOverride
		}
		u, err := url.Parse(fmt.Sprintf("https://%s/%s/%s", host, params.Owner, params.Project))
		if err != nil {
			return nil, errs.Wrap(err, "url parse new project url failed")
		}
		q := u.Query()

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

	languageDisabled := os.Getenv(constants.DisableLanguageTemplates) == "true"
	content := params.Content
	if !languageDisabled && content == "" && lang != language.Unset && lang != language.Unknown {
		tplName := "activestate.yaml." + strings.TrimRight(lang.String(), "23") + ".tpl"
		template, err := assets.ReadFileBytes(tplName)
		if err != nil {
			return nil, errs.Wrap(err, "Could not read asset")
		}
		content, err = strutils.ParseTemplate(
			string(template),
			map[string]interface{}{"Owner": owner, "Project": project, "Shell": shell, "Language": lang.String(), "LangExe": lang.Executable().Filename()},
			nil)
		if err != nil {
			return nil, errs.Wrap(err, "Could not parse %s", tplName)
		}
	}

	data := map[string]interface{}{
		"Project":       params.ProjectURL,
		"Content":       content,
		"Private":       params.Private,
		"ConfigVersion": ConfigVersion,
	}

	tplName := "activestate.yaml.tpl"
	tplContents, err := assets.ReadFileBytes(tplName)
	if err != nil {
		return nil, errs.Wrap(err, "Could not read asset")
	}
	fileContents, err := strutils.ParseTemplate(string(tplContents), data, nil)
	if err != nil {
		return nil, errs.Wrap(err, "Could not parse %s", tplName)
	}

	err = fileutils.WriteFile(params.path, []byte(fileContents))
	if err != nil {
		return nil, err
	}

	if params.Cache != "" {
		createErr := createHostFile(params.Directory, params.Cache, params.Portable)
		if createErr != nil {
			return nil, errs.Wrap(createErr, "Could not create cache file")
		}
	}

	return Parse(params.path)
}

func createHostFile(filePath, cachePath string, portable bool) error {
	user, err := user.Current()
	if err != nil {
		return errs.Wrap(err, "Could not get current user")
	}

	data := map[string]interface{}{
		"Cache":    cachePath,
		"Portable": portable,
	}

	tplName := "activestate.yaml.cache.tpl"
	tplContents, err := assets.ReadFileBytes(tplName)
	if err != nil {
		return errs.Wrap(err, "Could not read asset")
	}

	fileContents, err := strutils.ParseTemplate(string(tplContents), data, nil)
	if err != nil {
		return errs.Wrap(err, "Could not parse %s", tplName)
	}

	// Trim any non-alphanumeric characters from the username
	if err := fileutils.WriteFile(filepath.Join(filePath, fmt.Sprintf("activestate.%s.yaml", nonAlphanumericRegex.ReplaceAllString(user.Username, ""))), []byte(fileContents)); err != nil {
		return errs.Wrap(err, "Could not write cache file")
	}

	return nil
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

	dat, err := os.ReadFile(projectFilePath)
	if err != nil {
		return nil, errs.Wrap(err, "os.ReadFile %s failed", projectFilePath)
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
		Channel: split[0],
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
		return os.WriteFile(projectFilePath, replaced, 0644)
	}

	projectRegex := regexp.MustCompile(fmt.Sprintf("(?m:(^project:\\s*%s))", ProjectURLRe))
	lockString := fmt.Sprintf("%s@%s", branch, version)
	lockUpdate := []byte(fmt.Sprintf("${1}\nlock: %s", lockString))

	data, err = os.ReadFile(projectFilePath)
	if err != nil {
		return err
	}

	updated := projectRegex.ReplaceAll(data, lockUpdate)

	return os.WriteFile(projectFilePath, updated, 0644)
}

func RemoveLockInfo(projectFilePath string) error {
	data, err := os.ReadFile(projectFilePath)
	if err != nil {
		return locale.WrapError(err, "err_read_projectfile", "", projectFilePath)
	}

	lockRegex := regexp.MustCompile(`(?m)^lock:.*`)
	clean := lockRegex.ReplaceAll(data, []byte(""))

	err = os.WriteFile(projectFilePath, clean, 0644)
	if err != nil {
		return locale.WrapError(err, "err_write_unlocked_projectfile", "Could not remove lock from projectfile")
	}

	return nil
}

func cleanVersionInfo(projectFilePath string) ([]byte, error) {
	data, err := os.ReadFile(projectFilePath)
	if err != nil {
		return nil, locale.WrapError(err, "err_read_projectfile", "", projectFilePath)
	}

	branchRegex := regexp.MustCompile(`(?m:^branch:\s*\w+\n)`)
	clean := branchRegex.ReplaceAll(data, []byte(""))

	versionRegex := regexp.MustCompile(`(?m:^version:\s*\d+.\d+.\d+-[A-Za-z0-9]+\n)`)
	clean = versionRegex.ReplaceAll(clean, []byte(""))

	err = os.WriteFile(projectFilePath, clean, 0644)
	if err != nil {
		return nil, locale.WrapError(err, "err_write_clean_projectfile", "Could not write cleaned projectfile information")
	}

	return clean, nil
}

type ConfigGetter interface {
	GetStringMapStringSlice(key string) map[string][]string
	AllKeys() []string
	GetStringSlice(string) []string
	GetString(string) string
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

// GetStaleProjectMapping returns a project mapping from the last time the
// state tool was run. This mapping could include projects that are no longer
// on the system.
func GetStaleProjectMapping(config ConfigGetter) map[string][]string {
	addDeprecatedProjectMappings(config)
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
		if strings.EqualFold(key, namespace) {
			paths = append(paths, value...)
		}
	}

	return paths
}

// StoreProjectMapping associates the namespace with the project
// path in the config
func StoreProjectMapping(cfg ConfigGetter, namespace, projectPath string) {
	SetRecentlyUsedNamespace(cfg, namespace)
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
		multilog.Error("Could not set project mapping in config, error: %v", errs.JoinMessage(err))
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
					configFile := filepath.Join(path, constants.ConfigFileName)
					if !fileutils.DirExists(path) || !fileutils.FileExists(configFile) {
						removals = append(removals, i)
						continue
					}
					// Only remove the project if the activestate.yaml is parseable and there is a namespace
					// mismatch.
					// (We do not want to punish anyone for a syntax error when manually editing the file.)
					if proj, err := parse(configFile); err == nil && proj.Init() == nil {
						projNamespace := fmt.Sprintf("%s/%s", proj.Owner(), proj.Name())
						if namespace != projNamespace {
							removals = append(removals, i)
						}
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

func SetRecentlyUsedNamespace(cfg ConfigGetter, namespace string) {
	err := cfg.Set(constants.LastUsedNamespacePrefname, namespace)
	if err != nil {
		logging.Debug("Could not set recently used namespace in config, error: %v", err)
	}
}
