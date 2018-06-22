package project

import (
	"github.com/ActiveState/cli/internal/constraints"

	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/internal/variables"
	"github.com/ActiveState/cli/pkg/project"
	"github.com/ActiveState/cli/pkg/projectfile"
)

// Project covers the platform structure
type Project struct {
	projectfile  projectfile.Project
	platforms    []*projectfile.Platform
	languages    []*projectfile.Language
	variables    []*projectfile.Variable
	hooks        []*projectfile.Hook
	commands     []*projectfile.Command
}

//Platforms returns a reference to projectfile.Platforms
func (p Project) Platforms() []*Platform {
	var platforms []*Platform
	for platform := range p.projectfile.Platforms {
		var newPlat Platform
		newPlat.platform = *platform
		platforms = append(platforms, newPlat)
	}
	return platforms
}

//Languages returns a reference to projectfile.Languages
func (p Project) Languages() []*Language {
	var languages []*Language
	for language := range p.projectfile.Languages {
		var newLang Language
		newLang.language = *language
		languages = append(languages, newLang)
	}
	return languages
} 

//Variables returns a reference to projectfile.Variables
func (p Project) Variables() []*Variable {
	var variables []*Variable
	for variable := range p.projectfile.Variables {
		var newVar Variable
		newVar.variable = *variable
		variables = append(variables, newVar)
	}
	return variables
}

//Hooks returns a reference to projectfile.Hooks
func (p Project) Hooks() []*Hook {
	var hooks []*Hook
	for hook := range p.projectfile.Hooks {
		var newHook Hook
		newHook.hook = *hook
		hooks = append(hooks, newHook)
	}
	return hooks
}

//Commands returns a reference to projectfile.Commands
func (p Project) Commands() []*Command {
	var commands []*Command
	for command := range p.projectfile.Commands {
		var newCommand Command
		newCommand.command = *command
		commands = append(commands, newCommand)
	}
	return commands
}

//Name returned are contrained and all variables evaluated
func (p Project) Name() string { return p.platform.Name }

//Os returned are contrained and all variables evaluated
func (p Project) Os() string { return p.platform.Os }

//Version returned are contrained and all variables evaluated
func (p Project) Version() string { return p.platform.Version }

//Architecture returned are contrained and all variables evaluated
func (p Project) Architecture() string { return p.platform.Architecture }

//Libc returned are contrained and all variables evaluated
func (p Project) Libc() string { return p.platform.Libc }

//Compiler returned are contrained and all variables evaluated
func (p Project) Compiler() string { return p.platform.Compiler }

//Get returns project struct
func Get() *Project {
	pj := projectfile.Get()
	return Project{
		projectfile:  pj
	}
}

// Platform covers the platform structure
type Platform struct {
	platform *projectfile.Platform
}

//Name returned are contrained and all variables evaluated
func (p Platform) Name() string { return p.platform.Name }

//Os returned are contrained and all variables evaluated
func (p Platform) Os() string { return p.platform.Os }

//Version returned are contrained and all variables evaluated
func (p Platform) Version() string { return p.platform.Version }

//Architecture returned are contrained and all variables evaluated
func (p Platform) Architecture() string { return p.platform.Architecture }

//Libc returned are contrained and all variables evaluated
func (p Platform) Libc() string { return p.platform.Libc }

//Compiler returned are contrained and all variables evaluated
func (p Platform) Compiler() string { return p.platform.Compiler }

// Language covers the language structure, which goes under Project
type Language struct {
	build    *projectfile.Build
	language  *projectfile.Language
	packages []*projectfile.Package
}

//Name returned are contrained and all variables evaluated
func (l Language) Name() string { return l.language.Name }

//Version returned are contrained and all variables evaluated
func (l Language) Version() string { return l.language.Version }

//Build returned are contrained and all variables evaluated
func (l Language) Build() string { return l.build }

//Packages returned are contrained and all variables evaluated
func (l *Language) Packages() ([]Package, *failures.Failure) {
	validPackages := []Package{}
	for _, pkg := range l.packages {
		if !constraints.IsConstrained(pkg.Constraints) {
			newPkg := Package{}
			newPkg.package = pkg
			validPackages = append(validPackages, newPkg)
		}
	}
	return validPackages, nil
}

// Package covers the package structure, which goes under the language struct
type Package struct {
	pkg 	*projectfile.Package
	build   *projectfile.Build
}

//Name returned are contrained and all variables evaluated
func (p Package) Name() string { return p.pkg.Name }

//Version returned are contrained and all variables evaluated
func (p Package) Version() string { return p.pkg.Version }

//Build returned are contrained and all variables evaluated
func (p Package) Build() string { return p.pkg.build }

//Constraints returned are contrained and all variables evaluated
func (p Constraint) Constraints() string {
	validConstraints := []Constraint{}
	for _, constraint := range p.constraints {
		if !constraints.IsConstrained(constraint.Constraints) {
			newConstraint := Constraint{}
			newConstraint.constraint = constraint
			validConstraints = append(validConstraints, newConstraint)
		}
	}
	return validConstraints, nil
}

// Constraint covers the constraint structure, which can go under almost any other struct
type Constraint struct {
	constraint  *projectfile.Constraint
	Platform    string
	Environment string
}

//Platform returned are contrained and all variables evaluated
func (c Constraint) Platform() string { return p.pkg.Build }

//Environment returned are contrained and all variables evaluated
func (c Constraint) Environment() string { return p.pkg.Environment }

// Variable covers the variable structure, which goes under Project
type Variable struct {
	variable  *projectfile.Variable
	Name  string
	Value string
}

// Hook covers the hook structure, which goes under Project
type Hook struct {
	hook  *projectfile.Hook
	Name  string
	Value string
}

// Command covers the command structure, which goes under Project
type Command struct {
	command  *projectfile.Command
	Name       string
	Value      string
	Standalone bool
}

// FailProjectNotLoaded identifies a failure as being due to a missing project file
var FailProjectNotLoaded = failures.Type("project.fail.notparsed", failures.FailUser)

func (p Project) Platforms() ([]*project.Platform, *failures.Failure) {
	return Platform
}

//Name returned are contrained and all variables evaluated
func Name() (string, *failures.Failure) {
	pj, err := projectfile.GetSafe()
	if err != nil {
		return "", err
	}
	return pj.Name, err
}

//Owner returned are contrained and all variables evaluated
func Owner() (string, *failures.Failure) {
	pj, err := projectfile.GetSafe()
	if err != nil {
		return "", err
	}
	return pj.Owner, err
}

//Namespace returned are contrained and all variables evaluated
func Namespace() (string, *failures.Failure) {
	pj, err := projectfile.GetSafe()
	if err != nil {
		return "", err
	}
	return pj.Namespace, err
}

//Version returned are contrained and all variables evaluated
func Version() (string, *failures.Failure) {
	pj, err := projectfile.GetSafe()
	if err != nil {
		return "", err
	}
	return pj.Version, err
}

//Environment returned are contrained and all variables evaluated
func Environment() (string, *failures.Failure) {
	pj, err := projectfile.GetSafe()
	if err != nil {
		return "", err
	}
	return pj.Environments, err
}

//Platforms returned are contrained and all variables evaluated
func Platforms() ([]projectfile.Platform, *failures.Failure) {
	pj, err := projectfile.GetSafe()
	if err != nil {
		return nil, err
	}
	return pj.Platforms, err
}

//Hooks returned are contrained and all variables evaluated
func Hooks() ([]Hook, *failures.Failure) {
	pj, fail := projectfile.GetSafe()
	if fail != nil {
		return nil, fail
	}
	validHooks := []Hook{}
	for _, hook := range pj.Hooks {
		if !constraints.IsConstrained(hook.Constraints) {
			newHook := Hook{}
			newHook.Name = hook.Name
			value, fail := variables.ExpandFromProject(hook.Value, pj)
			if fail.ToError() != nil {
				return nil, fail
			}
			newHook.Value = value
			validHooks = append(validHooks, newHook)
		}
	}
	return validHooks, fail
}

//Languages returned are contrained and all variables evaluated
func Languages() ([]Language, *failures.Failure) {
	pj, err := projectfile.GetSafe()
	if err != nil {
		return nil, err
	}
	validlangs := []Language{}
	for _, language := range pj.Languages {
		if !constraints.IsConstrained(language.Constraints) {
			newLang := Language{}
			newLang.Name = language.Name
			newLang.Version = language.Version
			newLang.Build = language.Build
			newLang.packages = language.Packages
			validlangs = append(validlangs, newLang)
		}
	}
	return validlangs, err
}

//Commands returned are contrained and all variables evaluated
func Commands() ([]Command, *failures.Failure) {
	pj, err := projectfile.GetSafe()
	if err != nil {
		return nil, err
	}
	validCmds := []Command{}
	for _, command := range pj.Commands {
		if !constraints.IsConstrained(command.Constraints) {
			newCmd := Command{}
			newCmd.Name = command.Name
			value, fail := variables.ExpandFromProject(command.Value, pj)
			if fail.ToError() != nil {
				return nil, fail
			}
			newCmd.Value = value
			newCmd.Standalone = command.Standalone
			validCmds = append(validCmds, newCmd)
		}
	}
	return validCmds, err
}

//Variables returned are contrained and all variables evaluated
func Variables() ([]Variable, *failures.Failure) {
	pj, err := projectfile.GetSafe()
	if err != nil {
		return nil, err
	}
	validVars := []Variable{}
	for _, variable := range pj.Variables {
		if !constraints.IsConstrained(variable.Constraints) {
			newVar := Variable{}
			newVar.Name = variable.Name
			newVar.Value = variable.Value
			validVars = append(validVars, newVar)
		}
	}
	return validVars, err
}
