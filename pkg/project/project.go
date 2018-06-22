package project

import (
	"github.com/ActiveState/cli/internal/constraints"

	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/internal/variables"
	"github.com/ActiveState/cli/pkg/projectfile"
)

// FailProjectNotLoaded identifies a failure as being due to a missing project file
var FailProjectNotLoaded = failures.Type("project.fail.notparsed", failures.FailUser)

// Project covers the platform structure
type Project struct {
	projectfile *projectfile.Project
}

//Platforms returns a reference to projectfile.Platforms
func (p *Project) Platforms() []*Platform {
	var platforms []*Platform
	for _, platform := range p.projectfile.Platforms {
		var newPlat = Platform{}
		newPlat.platform = &platform
		platforms = append(platforms, &newPlat)
	}
	return platforms
}

//Languages returns a reference to projectfile.Languages
func (p *Project) Languages() []*Language {
	var languages []*Language
	for _, language := range p.projectfile.Languages {
		if !constraints.IsConstrained(language.Constraints) {
			var newLang = Language{}
			newLang.language = &language
			newLang.packages = &language.Packages
			languages = append(languages, &newLang)
		}
	}
	return languages
}

//Variables returns a reference to projectfile.Variables
func (p *Project) Variables() []*Variable {
	var variables []*Variable
	for _, variable := range p.projectfile.Variables {
		if !constraints.IsConstrained(variable.Constraints) {
			var newVar = Variable{}
			newVar.variable = &variable
			variables = append(variables, &newVar)
		}
	}
	return variables
}

//Hooks returns a reference to projectfile.Hooks
func (p *Project) Hooks() []*Hook {
	var hooks []*Hook
	for _, hook := range p.projectfile.Hooks {
		if !constraints.IsConstrained(hook.Constraints) {
			var newHook = Hook{}
			newHook.hook = &hook
			hooks = append(hooks, &newHook)
		}
	}
	return hooks
}

//Commands returns a reference to projectfile.Commands
func (p *Project) Commands() []*Command {
	var commands []*Command
	for _, command := range p.projectfile.Commands {
		if !constraints.IsConstrained(command.Constraints) {
			var newCommand = Command{}
			newCommand.command = &command
			commands = append(commands, &newCommand)
		}
	}
	return commands
}

//Name returned are contrained and all variables evaluated
func (p *Project) Name() string { return p.projectfile.Name }

//Owner returned are contrained and all variables evaluated
func (p *Project) Owner() string { return p.projectfile.Owner }

//Version returned are contrained and all variables evaluated
func (p *Project) Version() string { return p.projectfile.Version }

//Namespace returned are contrained and all variables evaluated
func (p *Project) Namespace() string { return p.projectfile.Namespace }

//Environments returned are contrained and all variables evaluated
func (p *Project) Environments() string { return p.projectfile.Environments }

//Get returns project struct
func Get() *Project {
	pj := projectfile.Get()
	return &Project{pj}
}

//GetSafe returns project struct
func GetSafe() (*Project, *failures.Failure) {
	pj, fail := projectfile.GetSafe()
	if fail.ToError() != nil {
		return nil, fail
	}
	return &Project{pj}, nil
}

// Platform covers the platform structure
type Platform struct {
	platform *projectfile.Platform
}

//Name returned are contrained and all variables evaluated
func (p *Platform) Name() string { return p.platform.Name }

//Os returned are contrained and all variables evaluated
func (p *Platform) Os() (string, *failures.Failure) {
	pj, fail := projectfile.GetSafe()
	if fail.ToError() != nil {
		return "", fail
	}
	value, fail := variables.ExpandFromProject(p.platform.Os, pj)
	if fail.ToError() != nil {
		return "", fail
	}
	return value, nil
}

//Version returned are contrained and all variables evaluated
func (p *Platform) Version() (string, *failures.Failure) {
	pj, fail := projectfile.GetSafe()
	if fail.ToError() != nil {
		return "", fail
	}
	value, fail := variables.ExpandFromProject(p.platform.Version, pj)
	if fail.ToError() != nil {
		return "", fail
	}
	return value, nil
}

//Architecture returned are contrained and all variables evaluated
func (p *Platform) Architecture() (string, *failures.Failure) {
	pj, fail := projectfile.GetSafe()
	if fail.ToError() != nil {
		return "", fail
	}
	value, fail := variables.ExpandFromProject(p.platform.Architecture, pj)
	if fail.ToError() != nil {
		return "", fail
	}
	return value, nil
}

//Libc returned are contrained and all variables evaluated
func (p *Platform) Libc() (string, *failures.Failure) {
	pj, fail := projectfile.GetSafe()
	if fail.ToError() != nil {
		return "", fail
	}
	value, fail := variables.ExpandFromProject(p.platform.Libc, pj)
	if fail.ToError() != nil {
		return "", fail
	}
	return value, nil
}

//Compiler returned are contrained and all variables evaluated
func (p *Platform) Compiler() (string, *failures.Failure) {
	pj, fail := projectfile.GetSafe()
	if fail.ToError() != nil {
		return "", fail
	}
	value, fail := variables.ExpandFromProject(p.platform.Compiler, pj)
	if fail.ToError() != nil {
		return "", fail
	}
	return value, nil
}

// Language covers the language structure, which goes under Project
type Language struct {
	build    *projectfile.Build
	language *projectfile.Language
	packages *[]projectfile.Package
}

//Name returned are contrained and all variables evaluated
func (l *Language) Name() string { return l.language.Name }

//Version returned are contrained and all variables evaluated
func (l *Language) Version() string { return l.language.Version }

//Build returned are contrained and all variables evaluated
func (l *Language) Build() *projectfile.Build { return l.build }

//Packages returned are contrained and all variables evaluated
func (l *Language) Packages() ([]Package, *failures.Failure) {
	validPackages := []Package{}
	for _, pkg := range *l.packages {
		if !constraints.IsConstrained(pkg.Constraints) {
			newPkg := Package{}
			newPkg.pkg = &pkg
			validPackages = append(validPackages, newPkg)
		}
	}
	return validPackages, nil
}

// Package covers the package structure, which goes under the language struct
type Package struct {
	pkg *projectfile.Package
}

//Name returned are contrained and all variables evaluated
func (p *Package) Name() string { return p.pkg.Name }

//Version returned are contrained and all variables evaluated
func (p *Package) Version() string { return p.pkg.Version }

//Build returned are contrained and all variables evaluated
func (p *Package) Build() *projectfile.Build { return &p.pkg.Build }

//Constraints returned are contrained and all variables evaluated
func (p *Package) Constraints() *projectfile.Constraint {
	return &p.pkg.Constraints
}

// Constraint covers the constraint structure, which can go under almost any other struct
type Constraint struct {
	constraint *projectfile.Constraint
}

//Platform returned are contrained and all variables evaluated
func (c *Constraint) Platform() string { return c.constraint.Platform }

//Environment returned are contrained and all variables evaluated
func (c *Constraint) Environment() string { return c.constraint.Environment }

// Variable covers the variable structure, which goes under Project
type Variable struct {
	variable *projectfile.Variable
}

//Name returned are contrained and all variables evaluated
func (v *Variable) Name() string { return v.variable.Name }

//Value returned are contrained and all variables evaluated
func (v *Variable) Value() (string, *failures.Failure) {
	pj, fail := projectfile.GetSafe()
	if fail.ToError() != nil {
		return "", fail
	}
	value, fail := variables.ExpandFromProject(v.variable.Value, pj)
	if fail.ToError() != nil {
		return "", fail
	}
	return value, nil
}

// Hook covers the hook structure, which goes under Project
type Hook struct {
	hook *projectfile.Hook
}

//Name returned are contrained and all variables evaluated
func (h *Hook) Name() string { return h.hook.Name }

//Value returned are contrained and all variables evaluated
func (h *Hook) Value() (string, *failures.Failure) {
	pj, fail := projectfile.GetSafe()
	if fail.ToError() != nil {
		return "", fail
	}
	value, fail := variables.ExpandFromProject(h.hook.Value, pj)
	if fail.ToError() != nil {
		return "", fail
	}
	return value, nil
}

// Command covers the command structure, which goes under Project
type Command struct {
	command *projectfile.Command
}

//Name returned are contrained and all variables evaluated
func (c *Command) Name() string { return c.command.Name }

//Value returned are contrained and all variables evaluated
func (c *Command) Value() (string, *failures.Failure) {
	pj, fail := projectfile.GetSafe()
	if fail.ToError() != nil {
		return "", fail
	}
	value, fail := variables.ExpandFromProject(c.command.Value, pj)
	if fail.ToError() != nil {
		return "", fail
	}
	return value, nil
}

//Standalone returns if the command is standalone or not
func (c *Command) Standalone() bool { return c.command.Standalone }
