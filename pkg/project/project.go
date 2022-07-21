package project

import (
	"errors"
	"fmt"
	"log"
	"os"
	"regexp"
	"runtime"
	"strings"

	"github.com/go-openapi/strfmt"

	"github.com/ActiveState/cli/internal/constraints"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/keypairs"
	"github.com/ActiveState/cli/internal/language"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/multilog"
	"github.com/ActiveState/cli/internal/osutils"
	"github.com/ActiveState/cli/internal/output"
	secretsapi "github.com/ActiveState/cli/pkg/platform/api/secrets"
	"github.com/ActiveState/cli/pkg/projectfile"
)

// Build covers the build structure
type Build map[string]string

var pConditional *constraints.Conditional
var normalizeRx *regexp.Regexp

func init() {
	var err error
	normalizeRx, err = regexp.Compile("[^a-zA-Z0-9]+")
	if err != nil {
		log.Panicf("normalizeRx: invalid regex: %v", err)
	}
}

// RegisterConditional is a a temporary method for registering our conditional as a global
// yes this is bad, but at the time of implementation refactoring the project package to not be global is out of scope
func RegisterConditional(conditional *constraints.Conditional) {
	pConditional = conditional
}

// Project covers the platform structure
type Project struct {
	projectfile *projectfile.Project
	output.Outputer
}

// Source returns the source projectfile
func (p *Project) Source() *projectfile.Project { return p.projectfile }

func (p *Project) SetCommit(commitID string) error {
	return p.Source().SetCommit(commitID, p.IsHeadless())
}

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
func (p *Project) Secrets(cfg keypairs.Configurable) []*Secret {
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
			secrets = append(secrets, p.NewSecret(s, SecretScopeUser, cfg))
		}
	}
	if p.projectfile.Secrets.Project != nil {
		constrained, err := constraints.FilterUnconstrained(pConditional, p.projectfile.Secrets.Project.AsConstrainedEntities())
		if err != nil {
			logging.Warning("Could not filter unconstrained project secrets: %v", err)
		}
		secs := projectfile.MakeSecretsFromConstrainedEntities(constrained)
		for _, secret := range secs {
			secrets = append(secrets, p.NewSecret(secret, SecretScopeProject, cfg))
		}
	}
	return secrets
}

// SecretByName returns a secret matching the given name (if any)
func (p *Project) SecretByName(name string, scope SecretScope, cfg keypairs.Configurable) *Secret {
	for _, secret := range p.Secrets(cfg) {
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
		events = append(events, &Event{e, p})
	}
	return events
}

// EventByName returns a reference to a projectfile.Script with a given name.
func (p *Project) EventByName(name string) *Event {
	for _, event := range p.Events() {
		if strings.ToLower(event.Name()) == strings.ToLower(name) {
			return event
		}
	}
	return nil
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
		scripts = append(scripts, &Script{s, p})
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

// Owner returns project owner
func (p *Project) Owner() string {
	return p.projectfile.Owner()
}

// Name returns project name
func (p *Project) Name() string {
	return p.projectfile.Name()
}

func (p *Project) Private() bool {
	return p.Source().Private
}

// CommitID returns project commitID
func (p *Project) CommitID() string {
	return p.projectfile.CommitID()
}

// CommitUUID returns project commitID in UUID format
func (p *Project) CommitUUID() strfmt.UUID {
	return strfmt.UUID(p.CommitID())
}

// BranchName returns the project branch name
func (p *Project) BranchName() string {
	return p.projectfile.BranchName()
}

// Path returns the project path
func (p *Project) Path() string {
	return p.projectfile.Path()
}

func (p *Project) IsHeadless() bool {
	match := projectfile.CommitURLRe.FindStringSubmatch(p.URL())
	return len(match) > 1
}

// NormalizedName returns the project name in a normalized format (alphanumeric, lowercase)
func (p *Project) NormalizedName() string {
	return strings.ToLower(normalizeRx.ReplaceAllString(p.Name(), ""))
}

// Version returns the locked state tool version
func (p *Project) Version() string { return p.projectfile.Version() }

// VersionBranch returns branch that we're pinned to (useless unless version is also set)
func (p *Project) VersionBranch() string { return p.projectfile.VersionBranch() }

// IsLocked returns whether the current project is locked
func (p *Project) IsLocked() bool { return p.Lock() != "" }

// Lock returns the lock information for this project
func (p *Project) Lock() string { return p.projectfile.Lock }

// Namespace returns project namespace
func (p *Project) Namespace() *Namespaced {
	commitID := strfmt.UUID(p.projectfile.CommitID())
	return &Namespaced{p.projectfile.Owner(), p.projectfile.Name(), &commitID, false}
}

// NamespaceString is a convenience function to make interfaces simpler
func (p *Project) NamespaceString() string {
	return p.Namespace().String()
}

// Environments returns project environment
func (p *Project) Environments() string { return p.projectfile.Environments }

// New creates a new Project struct
func New(p *projectfile.Project, out output.Outputer) (*Project, error) {
	project := &Project{projectfile: p, Outputer: out}
	return project, nil
}

// NewLegacy is for legacy use-cases only, DO NOT USE
func NewLegacy(p *projectfile.Project) (*Project, error) {
	return New(p, output.Get())
}

// Parse will parse the given projectfile and instantiate a Project struct with it
func Parse(fpath string) (*Project, error) {
	pjfile, err := projectfile.Parse(fpath)
	if err != nil {
		return nil, err
	}
	return New(pjfile, output.Get())
}

// Get returns project struct. Quits execution if error occurs
func Get() *Project {
	pj := projectfile.Get()
	project, err := New(pj, output.Get())
	if err != nil {
		fmt.Fprint(os.Stderr, locale.Tr("err_project_unavailable", err.Error()))
		os.Exit(1)
	}
	return project
}

// GetSafe returns project struct.  Produces failure if error occurs, allows recovery
func GetSafe() (*Project, error) {
	pjFile, err := projectfile.GetSafe()
	if err != nil {
		return nil, err
	}
	project, err := New(pjFile, output.Get())
	if err != nil {
		return nil, err
	}

	return project, nil
}

// GetOnce returns project struct the same as Get and GetSafe, but it avoids persisting the project
func GetOnce() (*Project, error) {
	wd, err := osutils.Getwd()
	if err != nil {
		return nil, errs.Wrap(err, "Getwd failure")
	}
	return FromPath(wd)
}

// FromPath will return the project that's located at the given path (this will walk up the directory tree until it finds the project)
func FromPath(path string) (*Project, error) {
	pjFile, err := projectfile.FromPath(path)
	if err != nil {
		return nil, err
	}
	project, err := New(pjFile, output.Get())
	if err != nil {
		return nil, err
	}

	return project, nil
}

// FromExactPath will return the project that's located at the given path without walking up the directory tree
func FromExactPath(path string) (*Project, error) {
	pjFile, err := projectfile.FromExactPath(path)
	if err != nil {
		return nil, err
	}
	project, err := New(pjFile, output.Get())
	if err != nil {
		return nil, err
	}

	return project, nil
}

// LocalProjectDoesNotExist is an error returned when a requested project is not checked out locally.
type LocalProjectDoesNotExist struct{ error }

// IsLocalProjectDoesNotExist checks if the error is a LocalProjectDoesNotExist.
func IsLocalProjectDoesNotExist(err error) bool {
	return errs.Matches(err, &LocalProjectDoesNotExist{})
}

// FromNamespaceLocal returns a local project (if any) that matches the given namespace.
// This is primarily used by `state use` in order to fetch a project to switch to if it already
// exists locally. The namespace may omit the owner.
func FromNamespaceLocal(ns *Namespaced, cfg projectfile.ConfigGetter) (*Project, error) {
	for namespace, paths := range projectfile.GetProjectMapping(cfg) {
		if len(paths) == 0 {
			continue
		}
		var namespaced Namespaced
		err := namespaced.Set(namespace)
		if err != nil {
			logging.Debug("Cannot parse namespace: %v") // should not happen since this is stored
			continue
		}
		if (!ns.AllowOmitOwner && strings.ToLower(namespaced.String()) == strings.ToLower(ns.String())) ||
			(ns.AllowOmitOwner && strings.ToLower(namespaced.Project) == strings.ToLower(ns.Project)) {
			return FromPath(paths[0]) // just pick the first one
		}
	}
	return nil, &LocalProjectDoesNotExist{errs.New("No local project found")}
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
	return Expand(p.platform.Os)
}

// Version returned with all secrets evaluated
func (p *Platform) Version() (string, error) {
	return Expand(p.platform.Version)
}

// Architecture with all secrets evaluated
func (p *Platform) Architecture() (string, error) {
	return Expand(p.platform.Architecture)
}

// Libc returned are constrained and all secrets evaluated
func (p *Platform) Libc() (string, error) {
	return Expand(p.platform.Libc)
}

// Compiler returned are constrained and all secrets evaluated
func (p *Platform) Compiler() (string, error) {
	return Expand(p.platform.Compiler)
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
		newVal, err := Expand(val)
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
		newVal, err := Expand(val)
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
	return Expand(c.constant.Value)
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
func NewSecretScope(name string) (SecretScope, error) {
	var scope SecretScope
	switch name {
	case string(SecretScopeUser):
		return SecretScopeUser, nil
	case string(SecretScopeProject):
		return SecretScopeProject, nil
	default:
		return scope, locale.NewInputError("secrets_err_invalid_namespace")
	}
}

// Secret covers the secret structure
type Secret struct {
	secret  *projectfile.Secret
	project *Project
	scope   SecretScope
	cfg     keypairs.Configurable
}

// InitSecret creates a new secret with the given name and all default settings
func (p *Project) InitSecret(name string, scope SecretScope, cfg keypairs.Configurable) *Secret {
	return p.NewSecret(&projectfile.Secret{
		Name: name,
	}, scope, cfg)
}

// NewSecret creates a new secret struct
func (p *Project) NewSecret(s *projectfile.Secret, scope SecretScope, cfg keypairs.Configurable) *Secret {
	return &Secret{s, p, scope, cfg}
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
func (s *Secret) ValueOrNil() (*string, error) {
	secretsExpander := NewSecretExpander(secretsapi.GetClient(), nil, nil, s.cfg)

	category := ProjectCategory
	if s.IsUser() {
		category = UserCategory
	}

	value, err := secretsExpander.Expand("", category, s.secret.Name, false, s.project)
	if err != nil {
		if errors.Is(err, ErrSecretNotFound) {
			return nil, nil
		}
		multilog.Error("Could not expand secret %s, error: %v", s.Name(), err)
		return nil, errs.Wrap(err, "secret for %s expansion failed", s.secret.Name)
	}
	return &value, nil
}

// Value returned with all secrets evaluated
func (s *Secret) Value() (string, error) {
	value, err := s.ValueOrNil()
	if err != nil || value == nil {
		return "", err
	}
	return *value, nil
}

// Event covers the hook structure
type Event struct {
	event   *projectfile.Event
	project *Project
}

// Source returns the source projectfile
func (e *Event) Source() *projectfile.Project { return e.project.projectfile }

// Name returns Event name
func (e *Event) Name() string { return e.event.Name }

// Value returned with all secrets evaluated
func (e *Event) Value() (string, error) {
	return Expand(e.event.Value)
}

// Scope returns the scope property of the event
func (e *Event) Scope() ([]string, error) {
	result := []string{}
	for _, s := range e.event.Scope {
		v, err := Expand(s)
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
}

// Source returns the source projectfile
func (script *Script) Source() *projectfile.Project { return script.project.projectfile }

// SourceScript returns the source script
func (script *Script) SourceScript() *projectfile.Script { return script.script }

// Name returns script name
func (script *Script) Name() string { return script.script.Name }

// Languages returns the languages of this script
func (script *Script) Languages() []language.Language {
	stringLanguages := strings.Split(script.script.Language, ",")
	languages := make([]language.Language, 0)
	for _, lang := range stringLanguages {
		if lang != "" {
			languages = append(languages, language.MakeByName(strings.TrimSpace(lang)))
		}
	}
	return languages
}

// LanguageSafe returns the first languages of this script. The
// returned languages are guaranteed to be of a known scripting language
func (script *Script) LanguageSafe() []language.Language {
	var langs []language.Language
	for _, lang := range script.Languages() {
		if !lang.Recognized() {
			continue
		}
		langs = append(langs, lang)
	}

	if len(langs) == 0 {
		return DefaultScriptLanguage()
	}

	return langs
}

// DefaultScriptLanguage returns the default script language for
// the current platform. (ie. batch or bash)
func DefaultScriptLanguage() []language.Language {
	if runtime.GOOS == "windows" {
		return []language.Language{language.Batch}
	}
	return []language.Language{language.Sh}
}

// Description returns script description
func (script *Script) Description() string { return script.script.Description }

// Value returned with all secrets evaluated
func (script *Script) Value() (string, error) {
	return Expand(script.script.Value)
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
