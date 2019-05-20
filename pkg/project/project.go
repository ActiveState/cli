package project

import (
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/ActiveState/cli/internal/constraints"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/secrets"
	mono_models "github.com/ActiveState/cli/pkg/platform/api/mono/mono_models"
	secretsapi "github.com/ActiveState/cli/pkg/platform/api/secrets"
	"github.com/ActiveState/cli/pkg/platform/model"

	"github.com/ActiveState/cli/internal/expander"
	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/pkg/projectfile"
)

// FailProjectNotLoaded identifies a failure as being due to a missing project file
var FailProjectNotLoaded = failures.Type("project.fail.notparsed", failures.FailUser)

// Build covers the build structure
type Build map[string]string

// Project covers the platform structure
type Project struct {
	projectfile *projectfile.Project
}

// Source returns the source projectfile
func (p *Project) Source() *projectfile.Project { return p.projectfile }

// Platforms gets platforms
func (p *Project) Platforms() []*Platform {
	platforms := []*Platform{}
	for i := range p.projectfile.Platforms {
		platforms = append(platforms, &Platform{&p.projectfile.Platforms[i], p.projectfile})
	}
	return platforms
}

// Languages returns a reference to projectfile.Languages
func (p *Project) Languages() []*Language {
	languages := []*Language{}
	for i, language := range p.projectfile.Languages {
		if !constraints.IsConstrained(language.Constraints) {
			languages = append(languages, &Language{&p.projectfile.Languages[i], p.projectfile})
		}
	}
	return languages
}

// Variables returns a reference to projectfile.Variables
func (p *Project) Variables() []*Variable {
	variables := []*Variable{}
	for i, variable := range p.projectfile.Variables {
		if !constraints.IsConstrained(variable.Constraints) {
			variables = append(variables, &Variable{p.projectfile.Variables[i], p.projectfile})
		}
	}
	return variables
}

// VariableByName returns a variable matching the given name (if any)
func (p *Project) VariableByName(name string) *Variable {
	for _, variable := range p.Variables() {
		if variable.Name() == name {
			return variable
		}
	}
	return nil
}

// Events returns a reference to projectfile.Events
func (p *Project) Events() []*Event {
	events := []*Event{}
	for i, event := range p.projectfile.Events {
		if !constraints.IsConstrained(event.Constraints) {
			events = append(events, &Event{&p.projectfile.Events[i], p.projectfile})
		}
	}
	return events
}

// Scripts returns a reference to projectfile.Scripts
func (p *Project) Scripts() []*Script {
	scripts := []*Script{}
	for i, script := range p.projectfile.Scripts {
		if !constraints.IsConstrained(script.Constraints) {
			scripts = append(scripts, &Script{&p.projectfile.Scripts[i], p.projectfile})
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

// Name returns project name
func (p *Project) Name() string { return p.projectfile.Name }

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

// Owner returns project owner
func (p *Project) Owner() string { return p.projectfile.Owner }

// Version returns project version
func (p *Project) Version() string { return p.projectfile.Version }

// Namespace returns project namespace
func (p *Project) Namespace() string { return p.projectfile.Namespace }

// Environments returns project environment
func (p *Project) Environments() string { return p.projectfile.Environments }

// New creates a new Project struct
func New(p *projectfile.Project) *Project {
	return &Project{p}
}

// Get returns project struct. Quits execution if error occurs
func Get() *Project {
	pj := projectfile.Get()
	return New(pj)
}

// GetSafe returns project struct.  Produces failure if error occurs, allows recovery
func GetSafe() (*Project, *failures.Failure) {
	pj, fail := projectfile.GetSafe()
	if fail.ToError() != nil {
		return nil, fail
	}
	return &Project{pj}, nil
}

// Platform covers the platform structure
type Platform struct {
	platform    *projectfile.Platform
	projectfile *projectfile.Project
}

// Source returns the source projectfile
func (p *Platform) Source() *projectfile.Project { return p.projectfile }

// Name returns platform name
func (p *Platform) Name() string { return p.platform.Name }

// Os returned with all variables evaluated
func (p *Platform) Os() string {
	value := expander.ExpandFromProject(p.platform.Os, p.projectfile)
	return value
}

// Version returned with all variables evaluated
func (p *Platform) Version() string {
	value := expander.ExpandFromProject(p.platform.Version, p.projectfile)
	return value
}

// Architecture with all variables evaluated
func (p *Platform) Architecture() string {
	value := expander.ExpandFromProject(p.platform.Architecture, p.projectfile)
	return value
}

// Libc returned are constrained and all variables evaluated
func (p *Platform) Libc() string {
	value := expander.ExpandFromProject(p.platform.Libc, p.projectfile)
	return value
}

// Compiler returned are constrained and all variables evaluated
func (p *Platform) Compiler() string {
	value := expander.ExpandFromProject(p.platform.Compiler, p.projectfile)
	return value
}

// Language covers the language structure
type Language struct {
	language    *projectfile.Language
	projectfile *projectfile.Project
}

// Source returns the source projectfile
func (l *Language) Source() *projectfile.Project { return l.projectfile }

// Name with all variables evaluated
func (l *Language) Name() string { return l.language.Name }

// Version with all variables evaluated
func (l *Language) Version() string { return l.language.Version }

// ID is an identifier for this language; e.g. the Name + Version
func (l *Language) ID() string {
	return l.Name() + l.Version()
}

// Build with all variables evaluated
func (l *Language) Build() *Build {
	build := Build{}
	for key, val := range l.language.Build {
		newVal := expander.ExpandFromProject(val, l.projectfile)
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
			newPkg.projectfile = l.projectfile
			validPackages = append(validPackages, newPkg)
		}
	}
	return validPackages
}

// Package covers the package structure
type Package struct {
	pkg         *projectfile.Package
	projectfile *projectfile.Project
}

// Source returns the source projectfile
func (p *Package) Source() *projectfile.Project { return p.projectfile }

// Name returns package name
func (p *Package) Name() string { return p.pkg.Name }

// Version returns package version
func (p *Package) Version() string { return p.pkg.Version }

// Build returned with all variables evaluated
func (p *Package) Build() *Build {
	build := Build{}
	for key, val := range p.pkg.Build {
		newVal := expander.ExpandFromProject(val, p.projectfile)
		build[key] = newVal
	}
	return &build
}

// Variable covers the variable structure
type Variable struct {
	variable    *projectfile.Variable
	projectfile *projectfile.Project
}

// Source returns the source projectfile
func (v *Variable) Source() *projectfile.Project { return v.projectfile }

// Name returns variable name
func (v *Variable) Name() string { return v.variable.Name }

// Description returns variable description
func (v *Variable) Description() string { return v.variable.Description }

// IsSecret returns whether this variable is a secret variable or static
func (v *Variable) IsSecret() bool { return v.variable.Value.StaticValue == nil }

// IsShared returns whether this variable is shared or not
func (v *Variable) IsShared() bool { return v.variable.Value.Share != nil }

// SharedWith returns who this variable is shared with
func (v *Variable) SharedWith() *projectfile.VariableShare { return v.variable.Value.Share }

// PulledFrom returns where this variable was pulled from
func (v *Variable) PulledFrom() *projectfile.VariablePullFrom { return v.variable.Value.PullFrom }

// ValueOrNil acts as Value() except it can return a nil
func (v *Variable) ValueOrNil() (*string, *failures.Failure) {
	variable := v.variable
	if variable.Value.StaticValue != nil {
		value := expander.ExpandFromProject(*variable.Value.StaticValue, v.projectfile)
		return &value, nil
	}

	secretsExpander := expander.NewSecretExpander(secretsapi.GetClient())
	value, failure := secretsExpander.Expand(v.variable, v.projectfile)
	if failure != nil {
		if failure.Type.Matches(secretsapi.FailUserSecretNotFound) {
			return nil, nil
		}
		logging.Error("Could not expand secret variable %s, error: %s", v.Name(), failure.Error())
		return nil, failure
	}
	return &value, nil
}

// StoreLabel returns a representation of the variable storage location.
func (v *Variable) StoreLabel() string {
	if !v.IsSecret() {
		return "local"
	}
	return v.PulledFrom().String()
}

// IsSetLabel returns a representation of whether the variable is set.
func (v *Variable) IsSetLabel() (string, *failures.Failure) {
	valornil, failure := v.ValueOrNil()
	if failure != nil {
		return "", failure
	}
	if valornil == nil {
		return locale.T("variables_value_unset"), nil
	}
	return locale.T("variables_value_set"), nil
}

// IsEncryptedLabel returns a representation of encryption status.
func (v *Variable) IsEncryptedLabel() string {
	if v.IsSecret() {
		return locale.T("confirmation")
	}
	return locale.T("contradiction")
}

// Value returned with all variables evaluated
func (v *Variable) Value() (string, *failures.Failure) {
	value, failure := v.ValueOrNil()
	if failure != nil || value == nil {
		return "", failure
	}
	return *value, nil
}

// Save will save the provided value for this variable to the project file if not a secret, else
// will store back to the secrets store.
func (v *Variable) Save(value string) *failures.Failure {
	if v.IsSecret() {
		return v.saveSecretValue(value)
	}
	return v.saveStaticValue(value)
}

func (v *Variable) saveSecretValue(value string) *failures.Failure {
	org, failure := model.FetchOrgByURLName(v.projectfile.Owner)
	if failure != nil {
		return failure
	}

	var project *mono_models.Project
	if projectfile.VariablePullFromProject == *v.PulledFrom() {
		project, failure = model.FetchProjectByName(org.Urlname, v.projectfile.Name)
		if failure != nil {
			return failure
		}
	}

	kp, failure := secrets.LoadKeypairFromConfigDir()
	if failure != nil {
		return failure
	}

	isShareable := v.IsShared() && projectfile.VariableShareOrg == *v.SharedWith()
	failure = secrets.Save(secretsapi.GetClient(), kp, org, project, !isShareable, v.Name(), value)
	if failure != nil {
		return failure
	} else if isShareable {
		return secrets.ShareWithOrgUsers(secretsapi.GetClient(), org, project, v.Name(), value)
	}

	return nil
}

func (v *Variable) saveStaticValue(value string) *failures.Failure {
	v.variable.ValueRaw = projectfile.VariableValue{StaticValue: &value}
	if err := v.projectfile.Save(); err != nil {
		return failures.FailIO.Wrap(err)
	}
	return nil
}

// Event covers the hook structure
type Event struct {
	event       *projectfile.Event
	projectfile *projectfile.Project
}

// Source returns the source projectfile
func (e *Event) Source() *projectfile.Project { return e.projectfile }

// Name returns Event name
func (e *Event) Name() string { return e.event.Name }

// Value returned with all variables evaluated
func (e *Event) Value() string {
	value := expander.ExpandFromProject(e.event.Value, e.projectfile)
	return value
}

// Script covers the command structure
type Script struct {
	script      *projectfile.Script
	projectfile *projectfile.Project
}

// Source returns the source projectfile
func (script *Script) Source() *projectfile.Project { return script.projectfile }

// Name returns script name
func (script *Script) Name() string { return script.script.Name }

// Description returns script description
func (script *Script) Description() string { return script.script.Description }

// Value returned with all variables evaluated
func (script *Script) Value() string {
	value := expander.ExpandFromProject(script.script.Value, script.projectfile)
	return value
}

// Standalone returns if the script is standalone or not
func (script *Script) Standalone() bool { return script.script.Standalone }
