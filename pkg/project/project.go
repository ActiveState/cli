package project

import (
	"fmt"
	"os"
	"regexp"
	"runtime"
	"strings"

	"github.com/ActiveState/cli/internal/constraints"
	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/internal/language"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/secrets"
	secretsapi "github.com/ActiveState/cli/pkg/platform/api/secrets"
	"github.com/ActiveState/cli/pkg/platform/model"
	"github.com/ActiveState/cli/pkg/projectfile"
)

// FailProjectNotLoaded identifies a failure as being due to a missing project file
var FailProjectNotLoaded = failures.Type("project.fail.notparsed", failures.FailUser)

// Build covers the build structure
type Build map[string]string

// Project covers the platform structure
type Project struct {
	projectfile *projectfile.Project
	owner       string
	name        string
	commitID    string
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
	languages := []*Language{}
	for i, language := range p.projectfile.Languages {
		if !constraints.IsConstrained(language.Constraints) {
			languages = append(languages, &Language{&p.projectfile.Languages[i], p})
		}
	}
	return languages
}

// Constants returns a reference to projectfile.Constants
func (p *Project) Constants() []*Constant {
	constants := []*Constant{}
	for i, constant := range p.projectfile.Constants {
		if !constraints.IsConstrained(constant.Constraints) {
			constants = append(constants, &Constant{p.projectfile.Constants[i], p})
		}
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
	if p.projectfile.Secrets != nil && p.projectfile.Secrets.User != nil {
		for _, secret := range p.projectfile.Secrets.User {
			if !constraints.IsConstrained(secret.Constraints) {
				secrets = append(secrets, p.NewSecret(secret, SecretScopeUser))
			}
		}
	}
	if p.projectfile.Secrets != nil && p.projectfile.Secrets.User != nil {
		for _, secret := range p.projectfile.Secrets.Project {
			if !constraints.IsConstrained(secret.Constraints) {
				secrets = append(secrets, p.NewSecret(secret, SecretScopeProject))
			}
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
	events := []*Event{}
	for i, event := range p.projectfile.Events {
		if !constraints.IsConstrained(event.Constraints) {
			events = append(events, &Event{&p.projectfile.Events[i], p})
		}
	}
	return events
}

// Scripts returns a reference to projectfile.Scripts
func (p *Project) Scripts() []*Script {
	scripts := []*Script{}
	for i, script := range p.projectfile.Scripts {
		if !constraints.IsConstrained(script.Constraints) {
			scripts = append(scripts, &Script{&p.projectfile.Scripts[i], p})
		}
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

// URL returns the Project field of the project file
func (p *Project) URL() string {
	return p.projectfile.Project
}

type urlMeta struct {
	owner    string
	name     string
	commitID string
}

func parseURL(url string) (*urlMeta, *failures.Failure) {
	fail := projectfile.ValidateProjectURL(url)
	if fail != nil {
		return nil, fail
	}
	match := projectfile.ProjectURLRe.FindStringSubmatch(url)
	parts := urlMeta{match[1], match[2], ""}
	if len(match) == 4 {
		parts.commitID = match[3]
	}
	return &parts, nil
}

// Owner returns project owner
func (p *Project) Owner() string {
	return p.owner
}

// Name returns project name
func (p *Project) Name() string {
	return p.name
}

// CommitID returns project commitID
func (p *Project) CommitID() string {
	return p.commitID
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

// Namespace returns project namespace
func (p *Project) Namespace() string { return p.projectfile.Namespace }

// Environments returns project environment
func (p *Project) Environments() string { return p.projectfile.Environments }

// New creates a new Project struct
func New(p *projectfile.Project) (*Project, *failures.Failure) {
	project := &Project{projectfile: p}
	parts, fail := parseURL(p.Project)
	if fail != nil {
		return nil, fail
	}
	project.owner = parts.owner
	project.name = parts.name
	project.commitID = parts.commitID
	return project, nil
}

// Get returns project struct. Quits execution if error occurs
func Get() *Project {
	pj := projectfile.Get()
	project, fail := New(pj)
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
	project, fail := New(pjFile)
	if fail != nil {
		return nil, fail
	}

	return project, nil
}

// GetOnce returns project struct the same as Get and GetSafe, but it avoids persisting the project
func GetOnce() (*Project, *failures.Failure) {
	pjFile, fail := projectfile.GetOnce()
	if fail != nil {
		return nil, fail
	}
	project, fail := New(pjFile)
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
func (p *Platform) Os() string {
	value := Expand(p.platform.Os)
	return value
}

// Version returned with all secrets evaluated
func (p *Platform) Version() string {
	value := Expand(p.platform.Version)
	return value
}

// Architecture with all secrets evaluated
func (p *Platform) Architecture() string {
	value := Expand(p.platform.Architecture)
	return value
}

// Libc returned are constrained and all secrets evaluated
func (p *Platform) Libc() string {
	value := Expand(p.platform.Libc)
	return value
}

// Compiler returned are constrained and all secrets evaluated
func (p *Platform) Compiler() string {
	value := Expand(p.platform.Compiler)
	return value
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
func (l *Language) Build() *Build {
	build := Build{}
	for key, val := range l.language.Build {
		newVal := Expand(val)
		build[key] = newVal
	}
	return &build
}

// Packages returned are constrained set
func (l *Language) Packages() []Package {
	validPackages := []Package{}
	for i, pkg := range l.language.Packages {
		if !constraints.IsConstrained(pkg.Constraints) {
			newPkg := Package{}
			newPkg.pkg = &l.language.Packages[i]
			newPkg.project = l.project
			validPackages = append(validPackages, newPkg)
		}
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
func (p *Package) Build() *Build {
	build := Build{}
	for key, val := range p.pkg.Build {
		newVal := Expand(val)
		build[key] = newVal
	}
	return &build
}

// Constant covers the constant structure
type Constant struct {
	constant *projectfile.Constant
	project  *Project
}

// Name returns constant name
func (c *Constant) Name() string { return c.constant.Name }

// Value returns constant name
func (c *Constant) Value() string {
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
	secretsExpander := NewSecretExpander(secretsapi.GetClient(), s.IsUser())

	value, fail := secretsExpander.Expand(s.secret.Name, s.project)
	if fail != nil {
		if fail.Type.Matches(secretsapi.FailUserSecretNotFound) {
			return nil, nil
		}
		logging.Error("Could not expand secret %s, error: %s", s.Name(), fail.Error())
		return nil, fail
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

	remoteProject, fail := model.FetchProjectByName(org.Urlname, s.project.Name())
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
}

// Source returns the source projectfile
func (e *Event) Source() *projectfile.Project { return e.project.projectfile }

// Name returns Event name
func (e *Event) Name() string { return e.event.Name }

// Value returned with all secrets evaluated
func (e *Event) Value() string {
	value := Expand(e.event.Value)
	return value
}

// Script covers the command structure
type Script struct {
	script  *projectfile.Script
	project *Project
}

// Source returns the source projectfile
func (script *Script) Source() *projectfile.Project { return script.project.projectfile }

// Name returns script name
func (script *Script) Name() string { return script.script.Name }

// Language returns the language of this script
func (script *Script) Language() language.Language {
	return script.script.Language
}

// LanguageSafe returns the language of this script. The returned
// language is guaranteed to be of a known scripting language
func (script *Script) LanguageSafe() language.Language {
	if script.Language() == language.Unknown {
		return defaultScriptLanguage()
	}
	return script.Language()
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
func (script *Script) Value() string {
	value := Expand(script.script.Value)
	return value
}

// Raw returns the script value with no secrets or constants expanded
func (script *Script) Raw() string {
	return script.script.Value
}

// Standalone returns if the script is standalone or not
func (script *Script) Standalone() bool { return script.script.Standalone }
