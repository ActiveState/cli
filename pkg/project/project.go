package project

import (
	"fmt"
	"os"
	"regexp"
	"runtime"
	"strings"

	"github.com/go-openapi/strfmt"

	"github.com/ActiveState/cli/internal/constraints"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/internal/language"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/osutils"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/prompt"
	"github.com/ActiveState/cli/internal/secrets"
	secretsapi "github.com/ActiveState/cli/pkg/platform/api/secrets"
	"github.com/ActiveState/cli/pkg/platform/model"
	"github.com/ActiveState/cli/pkg/projectfile"
)

// FailProjectNotLoaded identifies a failure as being due to a missing project file
var FailProjectNotLoaded = failures.Type("project.fail.notparsed", failures.FailUser)

// Build covers the build structure
type Build map[string]string

var pConditional *constraints.Conditional

// RegisterConditional is a a temporary method for registering our conditional as a global
// yes this is bad, but at the time of implementation refactoring the project package to not be global is out of scope
func RegisterConditional(conditional *constraints.Conditional) {
	pConditional = conditional
}

// Project covers the platform structure
type Project struct {
	projectfile *projectfile.Project
	owner       string
	name        string
	commitID    string
	output.Outputer
	prompt.Prompter
}

// Source returns the source projectfile
func (p *Project) Source() *projectfile.Project { return p.projectfile }

// Platforms gets platforms
func (p *Project) Platforms() []*Platform {
	platforms := []*Platform{}
	for i := range p.projectfile.Platforms {
		platforms = append(platforms, &Platform{&p.projectfile.Platforms[i], p})
	}
	return platforms
}

// Languages returns a reference to projectfile.Languages
func (p *Project) Languages() []*Language {
	constrained, err := constraints.FilterUnconstrained(pConditional, p.projectfile.Languages.AsConstrainedEntities())
	if err != nil {
		logging.Warning("Could not filter unconstrained languages: %v", err)
	}
	ls := projectfile.MakeLanguagesFromConstrainedEntities(constrained)
	languages := []*Language{}
	for _, l := range ls {
		languages = append(languages, &Language{l, p})
	}
	return languages
}

// Constants returns a reference to projectfile.Constants
func (p *Project) Constants() []*Constant {
	constrained, err := constraints.FilterUnconstrained(pConditional, p.projectfile.Constants.AsConstrainedEntities())
	if err != nil {
		logging.Warning("Could not filter unconstrained constants: %v", err)
	}
	cs := projectfile.MakeConstantsFromConstrainedEntities(constrained)
	constants := []*Constant{}
	for _, c := range cs {
		constants = append(constants, &Constant{c, p})
	}
	return constants
}

// ConstantByName returns a constant matching the given name (if any)
func (p *Project) ConstantByName(name string) *Constant {
	for _, constant := range p.Constants() {
		if constant.Name() == name {
			return constant
		}
	}
	return nil
}

// Secrets returns a reference to projectfile.Secrets
func (p *Project) Secrets() []*Secret {
	secrets := []*Secret{}
	if p.projectfile.Secrets == nil {
		return secrets
	}
	if p.projectfile.Secrets.User != nil {
		constrained, err := constraints.FilterUnconstrained(pConditional, p.projectfile.Secrets.User.AsConstrainedEntities())
		if err != nil {
			logging.Warning("Could not filter unconstrained user secrets: %v", err)
		}
		secs := projectfile.MakeSecretsFromConstrainedEntities(constrained)
		for _, s := range secs {
			secrets = append(secrets, p.NewSecret(s, SecretScopeUser))
		}
	}
	if p.projectfile.Secrets.Project != nil {
		constrained, err := constraints.FilterUnconstrained(pConditional, p.projectfile.Secrets.Project.AsConstrainedEntities())
		if err != nil {
			logging.Warning("Could not filter unconstrained project secrets: %v", err)
		}
		secs := projectfile.MakeSecretsFromConstrainedEntities(constrained)
		for _, secret := range secs {
			secrets = append(secrets, p.NewSecret(secret, SecretScopeProject))
		}
	}
	return secrets
}

// SecretByName returns a secret matching the given name (if any)
func (p *Project) SecretByName(name string, scope SecretScope) *Secret {
	for _, secret := range p.Secrets() {
		if secret.Name() == name && secret.scope == scope {
			return secret
		}
	}
	return nil
}

// Events returns a reference to projectfile.Events
func (p *Project) Events() []*Event {
	constrained, err := constraints.FilterUnconstrained(pConditional, p.projectfile.Events.AsConstrainedEntities())
	if err != nil {
		logging.Warning("Could not filter unconstrained events: %v", err)
	}
	es := projectfile.MakeEventsFromConstrainedEntities(constrained)
	events := make([]*Event, 0, len(es))
	for _, e := range es {
		events = append(events, &Event{e, p, p.Outputer, p.Prompter})
	}
	return events
}

// Scripts returns a reference to projectfile.Scripts
func (p *Project) Scripts() []*Script {
	constrained, err := constraints.FilterUnconstrained(pConditional, p.projectfile.Scripts.AsConstrainedEntities())
	if err != nil {
		logging.Warning("Could not filter unconstrained scripts: %v", err)
	}
	scs := projectfile.MakeScriptsFromConstrainedEntities(constrained)
	scripts := make([]*Script, 0, len(scs))
	for _, s := range scs {
		scripts = append(scripts, &Script{s, p, p.Outputer, p.Prompter})
	}
	return scripts
}

// ScriptByName returns a reference to a projectfile.Script with a given name.
func (p *Project) ScriptByName(name string) *Script {
	for _, script := range p.Scripts() {
		if script.Name() == name {
			return script
		}
	}
	return nil
}

// Jobs returns a reference to projectfile.Jobs
func (p *Project) Jobs() []*Job {
	jobs := []*Job{}
	for _, j := range p.projectfile.Jobs {
		jobs = append(jobs, &Job{&j, p})
	}
	return jobs
}

// URL returns the Project field of the project file
func (p *Project) URL() string {
	return p.projectfile.Project
}

type urlMeta struct {
	owner    string
	name     string
	commitID string
}

func parseURL(url string) (urlMeta, *failures.Failure) {
	fail := projectfile.ValidateProjectURL(url)
	if fail != nil {
		return urlMeta{}, fail
	}

	match := projectfile.CommitURLRe.FindStringSubmatch(url)
	if len(match) > 1 {
		parts := urlMeta{"", "", match[1]}
		return parts, nil
	}

	match = projectfile.ProjectURLRe.FindStringSubmatch(url)
	parts := urlMeta{match[1], match[2], ""}
	if len(match) == 4 {
		parts.commitID = match[3]
	}

	return parts, nil
}

// Owner returns project owner
func (p *Project) Owner() string {
	return p.owner
}

// Name returns project name
func (p *Project) Name() string {
	return p.name
}

func (p *Project) Private() bool {
	return p.Source().Private
}

// CommitID returns project commitID
func (p *Project) CommitID() string {
	return p.commitID
}

// CommitUUID returns project commitID in UUID format
func (p *Project) CommitUUID() (*strfmt.UUID, error) {
	if ok := strfmt.Default.Validates("uuid", p.commitID); !ok {
		return nil, locale.NewError("invalid_uuid_val", "Invalid commit ID {{.V0}} in activestate.yaml.  You could replace it with 'latest'", p.commitID)
	}

	var uuid strfmt.UUID
	if err := uuid.UnmarshalText([]byte(p.commitID)); err != nil {
		return nil, locale.WrapError(err, "err_commit_id_unmarshal", "Failed to unmarshal the commit id {{.V0}} read from activestate.yaml.", p.commitID)
	}

	return &uuid, nil
}

func (p *Project) IsHeadless() bool {
	match := projectfile.CommitURLRe.FindStringSubmatch(p.URL())
	return len(match) > 1
}

// NormalizedName returns the project name in a normalized format (alphanumeric, lowercase)
func (p *Project) NormalizedName() string {
	rx, err := regexp.Compile("[^a-zA-Z0-9]+")
	if err != nil {
		failures.Handle(err, fmt.Sprintf("Regex failed to compile, error: %v", err))

		// This should only happen while in development, hence the os.Exit
		os.Exit(1)
	}

	return strings.ToLower(rx.ReplaceAllString(p.Name(), ""))
}

// Version returns project version
func (p *Project) Version() string { return p.projectfile.Version }

// Branch returns branch that we're pinned to (useless unless version is also set)
func (p *Project) Branch() string { return p.projectfile.Branch }

// Lock returns the lock information for this project
func (p *Project) Lock() string { return p.projectfile.Lock }

// Namespace returns project namespace
func (p *Project) Namespace() *Namespaced {
	commitID := strfmt.UUID(p.commitID)
	return &Namespaced{p.owner, p.name, &commitID}
}

// Environments returns project environment
func (p *Project) Environments() string { return p.projectfile.Environments }

// New creates a new Project struct
func New(p *projectfile.Project, out output.Outputer, prompt prompt.Prompter) (*Project, *failures.Failure) {
	project := &Project{projectfile: p, Outputer: out, Prompter: prompt}
	parts, fail := parseURL(p.Project)
	if fail != nil {
		return nil, fail
	}
	project.owner = parts.owner
	project.name = parts.name
	project.commitID = parts.commitID
	return project, nil
}

// NewLegacy is for legacy use-cases only, DO NOT USE
func NewLegacy(p *projectfile.Project) (*Project, *failures.Failure) {
	return New(p, output.Get(), prompt.New())
}

// Parse will parse the given projectfile and instantiate a Project struct with it
func Parse(fpath string) (*Project, *failures.Failure) {
	pjfile, fail := projectfile.Parse(fpath)
	if fail != nil {
		return nil, fail
	}
	return New(pjfile, output.Get(), prompt.New())
}

// Get returns project struct. Quits execution if error occurs
func Get() *Project {
	pj := projectfile.Get()
	project, fail := New(pj, output.Get(), prompt.New())
	if fail != nil {
		failures.Handle(fail, locale.T("err_project_unavailable"))
		os.Exit(1)
	}
	return project
}

// GetSafe returns project struct.  Produces failure if error occurs, allows recovery
func GetSafe() (*Project, *failures.Failure) {
	pjFile, fail := projectfile.GetSafe()
	if fail != nil {
		return nil, fail
	}
	project, fail := New(pjFile, output.Get(), prompt.New())
	if fail != nil {
		return nil, fail
	}

	return project, nil
}

// GetOnce returns project struct the same as Get and GetSafe, but it avoids persisting the project
func GetOnce() (*Project, *failures.Failure) {
	wd, err := osutils.Getwd()
	if err != nil {
		return nil, failures.FailIO.Wrap(err)
	}
	return FromPath(wd)
}

// FromPath will return the project that's located at the given path (this will walk up the directory tree until it finds the project)
func FromPath(path string) (*Project, *failures.Failure) {
	pjFile, fail := projectfile.FromPath(path)
	if fail != nil {
		return nil, fail
	}
	project, fail := New(pjFile, output.Get(), prompt.New())
	if fail != nil {
		return nil, fail
	}

	return project, nil
}

// Platform covers the platform structure
type Platform struct {
	platform *projectfile.Platform
	project  *Project
}

// Source returns the source projectfile
func (p *Platform) Source() *projectfile.Project { return p.project.projectfile }

// Name returns platform name
func (p *Platform) Name() string { return p.platform.Name }

// Os returned with all secrets evaluated
func (p *Platform) Os() (string, error) {
	return Expand(p.platform.Os, p.project.Outputer, p.project.Prompter)
}

// Version returned with all secrets evaluated
func (p *Platform) Version() (string, error) {
	return Expand(p.platform.Version, p.project.Outputer, p.project.Prompter)
}

// Architecture with all secrets evaluated
func (p *Platform) Architecture() (string, error) {
	return Expand(p.platform.Architecture, p.project.Outputer, p.project.Prompter)
}

// Libc returned are constrained and all secrets evaluated
func (p *Platform) Libc() (string, error) {
	return Expand(p.platform.Libc, p.project.Outputer, p.project.Prompter)
}

// Compiler returned are constrained and all secrets evaluated
func (p *Platform) Compiler() (string, error) {
	return Expand(p.platform.Compiler, p.project.Outputer, p.project.Prompter)
}

// Language covers the language structure
type Language struct {
	language *projectfile.Language
	project  *Project
}

// Source returns the source projectfile
func (l *Language) Source() *projectfile.Project { return l.project.projectfile }

// Name with all secrets evaluated
func (l *Language) Name() string { return l.language.Name }

// Version with all secrets evaluated
func (l *Language) Version() string { return l.language.Version }

// ID is an identifier for this language; e.g. the Name + Version
func (l *Language) ID() string {
	return l.Name() + l.Version()
}

// Build with all secrets evaluated
func (l *Language) Build() (*Build, error) {
	build := Build{}
	for key, val := range l.language.Build {
		newVal, err := Expand(val, l.project.Outputer, l.project.Prompter)
		if err != nil {
			return nil, err
		}
		build[key] = newVal
	}
	return &build, nil
}

// Packages returned are constrained set
func (l *Language) Packages() []Package {
	constrained, err := constraints.FilterUnconstrained(pConditional, l.language.Packages.AsConstrainedEntities())
	if err != nil {
		logging.Warning("Could not filter unconstrained packages: %v", err)
	}
	ps := projectfile.MakePackagesFromConstrainedEntities(constrained)
	validPackages := make([]Package, 0, len(ps))
	for _, pkg := range ps {
		validPackages = append(validPackages, Package{pkg: pkg, project: l.project})
	}
	return validPackages
}

// Package covers the package structure
type Package struct {
	pkg     *projectfile.Package
	project *Project
}

// Source returns the source projectfile
func (p *Package) Source() *projectfile.Project { return p.project.projectfile }

// Name returns package name
func (p *Package) Name() string { return p.pkg.Name }

// Version returns package version
func (p *Package) Version() string { return p.pkg.Version }

// Build returned with all secrets evaluated
func (p *Package) Build() (*Build, error) {
	build := Build{}
	for key, val := range p.pkg.Build {
		newVal, err := Expand(val, p.project.Outputer, p.project.Prompter)
		if err != nil {
			return nil, err
		}
		build[key] = newVal
	}
	return &build, nil
}

// Constant covers the constant structure
type Constant struct {
	constant *projectfile.Constant
	project  *Project
}

// Name returns constant name
func (c *Constant) Name() string { return c.constant.Name }

// Value returns constant value
func (c *Constant) Value() (string, error) {
	return Expand(c.constant.Value, c.project.Outputer, c.project.Prompter)
}

// SecretScope defines the scope of a secret
type SecretScope string

func (s *SecretScope) toString() string {
	return string(*s)
}

const (
	// SecretScopeUser defines a secret as being a user secret
	SecretScopeUser SecretScope = "user"
	// SecretScopeProject defines a secret as being a Project secret
	SecretScopeProject SecretScope = "project"
)

// NewSecretScope creates a new SecretScope from the given string name and will fail if the given string name does not
// match one of the available scopes
func NewSecretScope(name string) (SecretScope, *failures.Failure) {
	var scope SecretScope
	switch name {
	case string(SecretScopeUser):
		return SecretScopeUser, nil
	case string(SecretScopeProject):
		return SecretScopeProject, nil
	default:
		return scope, failures.FailInput.New("secrets_err_invalid_namespace")
	}
}

// Secret covers the secret structure
type Secret struct {
	secret  *projectfile.Secret
	project *Project
	scope   SecretScope
}

// InitSecret creates a new secret with the given name and all default settings
func (p *Project) InitSecret(name string, scope SecretScope) *Secret {
	return p.NewSecret(&projectfile.Secret{
		Name: name,
	}, scope)
}

// NewSecret creates a new secret struct
func (p *Project) NewSecret(s *projectfile.Secret, scope SecretScope) *Secret {
	return &Secret{s, p, scope}
}

// Source returns the source projectfile
func (s *Secret) Source() *projectfile.Project { return s.project.projectfile }

// Name returns secret name
func (s *Secret) Name() string { return s.secret.Name }

// Description returns secret description
func (s *Secret) Description() string { return s.secret.Description }

// IsUser returns whether this secret is user scoped
func (s *Secret) IsUser() bool { return s.scope == SecretScopeUser }

// Scope returns the scope as a string
func (s *Secret) Scope() string { return s.scope.toString() }

// IsProject returns whether this secret is project scoped
func (s *Secret) IsProject() bool { return s.scope == SecretScopeProject }

// ValueOrNil acts as Value() except it can return a nil
func (s *Secret) ValueOrNil() (*string, *failures.Failure) {
	secretsExpander := NewSecretExpander(secretsapi.GetClient(), nil)

	category := ProjectCategory
	if s.IsUser() {
		category = UserCategory
	}

	value, err := secretsExpander.Expand(category, s.secret.Name, false, s.project)
	if err != nil {
		if errs.Matches(err, ErrSecretNotFound) {
			return nil, nil
		}
		logging.Error("Could not expand secret %s, error: %v", s.Name(), err)
		return nil, failures.FailMisc.Wrap(err)
	}
	return &value, nil
}

// Value returned with all secrets evaluated
func (s *Secret) Value() (string, *failures.Failure) {
	value, fail := s.ValueOrNil()
	if fail != nil || value == nil {
		return "", fail
	}
	return *value, nil
}

// Save will save the provided value for this secret to the project file if not a secret, else
// will store back to the secrets store.
func (s *Secret) Save(value string) *failures.Failure {
	org, fail := model.FetchOrgByURLName(s.project.Owner())
	if fail != nil {
		return fail
	}

	remoteProject, fail := model.FetchProjectByName(org.URLname, s.project.Name())
	if fail != nil {
		return fail
	}

	kp, fail := secrets.LoadKeypairFromConfigDir()
	if fail != nil {
		return fail
	}

	fail = secrets.Save(secretsapi.GetClient(), kp, org, remoteProject, s.IsUser(), s.Name(), value)
	if fail != nil {
		return fail
	}

	if s.IsProject() {
		return secrets.ShareWithOrgUsers(secretsapi.GetClient(), org, remoteProject, s.Name(), value)
	}

	return nil
}

// Event covers the hook structure
type Event struct {
	event   *projectfile.Event
	project *Project
	output.Outputer
	prompt.Prompter
}

// Source returns the source projectfile
func (e *Event) Source() *projectfile.Project { return e.project.projectfile }

// Name returns Event name
func (e *Event) Name() string { return e.event.Name }

// Value returned with all secrets evaluated
func (e *Event) Value() (string, error) {
	return Expand(e.event.Value, e.Outputer, e.Prompter)
}

// Scope returns the scope property of the event
func (e *Event) Scope() ([]string, error) {
	result := []string{}
	for _, s := range e.event.Scope {
		v, err := Expand(s, e.project.Outputer, e.project.Prompter)
		if err != nil {
			return result, err
		}
		result = append(result, v)
	}
	return result, nil
}

// Script covers the command structure
type Script struct {
	script  *projectfile.Script
	project *Project
	output.Outputer
	prompt.Prompter
}

// Source returns the source projectfile
func (script *Script) Source() *projectfile.Project { return script.project.projectfile }

// SourceScript returns the source script
func (script *Script) SourceScript() *projectfile.Script { return script.script }

// Name returns script name
func (script *Script) Name() string { return script.script.Name }

// Language returns the language of this script
func (script *Script) Language() language.Language {
	return script.script.Language
}

// LanguageSafe returns the language of this script. The returned
// language is guaranteed to be of a known scripting language
func (script *Script) LanguageSafe() language.Language {
	lang := script.Language()
	if !lang.Recognized() {
		return defaultScriptLanguage()
	}
	return lang
}

func defaultScriptLanguage() language.Language {
	if runtime.GOOS == "windows" {
		return language.Batch
	}
	return language.Sh
}

// Description returns script description
func (script *Script) Description() string { return script.script.Description }

// Value returned with all secrets evaluated
func (script *Script) Value() (string, error) {
	return Expand(script.script.Value, script.Outputer, script.Prompter)
}

// Raw returns the script value with no secrets or constants expanded
func (script *Script) Raw() string {
	return script.script.Value
}

// Standalone returns if the script is standalone or not
func (script *Script) Standalone() bool { return script.script.Standalone }

// cacheFile allows this script to have an associated file
func (script *Script) setCachedFile(filename string) {
	script.script.Filename = filename
}

// filename returns the name of the file associated with this script
func (script *Script) cachedFile() string {
	return script.script.Filename
}

// Job covers the command structure
type Job struct {
	job     *projectfile.Job
	project *Project
}

func (j *Job) Name() string {
	return j.job.Name
}

func (j *Job) Constants() []*Constant {
	constants := []*Constant{}
	for _, constantName := range j.job.Constants {
		if constant := j.project.ConstantByName(constantName); constant != nil {
			constants = append(constants, constant)
		}
	}
	return constants
}

func (j *Job) Scripts() []*Script {
	scripts := []*Script{}
	for _, scriptName := range j.job.Scripts {
		if script := j.project.ScriptByName(scriptName); script != nil {
			scripts = append(scripts, script)
		}
	}
	return scripts
}
