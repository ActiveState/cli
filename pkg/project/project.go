package project

import (
	"github.com/ActiveState/cli/internal/constraints"

	"github.com/ActiveState/cli/pkg/projectfile"

	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/internal/variables"
)

// Platform covers the platform structure of our yaml
type Project struct {
}

// Platform covers the platform structure of our yaml
type Platform struct {
	Name         string
	Os           string
	Version      string
	Architecture string
	Libc         string
	Compiler     string
}

// Language covers the language structure, which goes under Project
type Language struct {
	Name     string
	Version  string
	Build    projectfile.Build
	packages []projectfile.Package
}

// Constraint covers the constraint structure, which can go under almost any other struct
type Constraint struct {
	Platform    string
	Environment string
}

// Package covers the package structure, which goes under the language struct
type Package struct {
	Name    string
	Version string
	Build   projectfile.Build
}

// Variable covers the variable structure, which goes under Project
type Variable struct {
	Name  string
	Value string
}

// Hook covers the hook structure, which goes under Project
type Hook struct {
	Name  string
	Value string
}

// Command covers the command structure, which goes under Project
type Command struct {
	Name       string
	Value      string
	Standalone bool
}

// FailProjectNotLoaded identifies a failure as being due to a missing project file
var FailProjectNotLoaded = failures.Type("fail.notparsed", failures.FailUser)

//Name returned are contrained and all variables evaluated
func (p *Project) Name() (string, error) {
	pj, err := projectfile.GetSafe()
	if err != nil {
		return "", err
	}
	return pj.Name, err
}

//Owner returned are contrained and all variables evaluated
func (p *Project) Owner() (string, error) {
	pj, err := projectfile.GetSafe()
	if err != nil {
		return "", err
	}
	return pj.Owner, err
}

//Namespace returned are contrained and all variables evaluated
func (p *Project) Namespace() (string, error) {
	pj, err := projectfile.GetSafe()
	if err != nil {
		return "", err
	}
	return pj.Namespace, err
}

//Version returned are contrained and all variables evaluated
func (p *Project) Version() (string, error) {
	pj, err := projectfile.GetSafe()
	if err != nil {
		return "", err
	}
	return pj.Version, err
}

//Environment returned are contrained and all variables evaluated
func (p *Project) Environment() (string, error) {
	pj, err := projectfile.GetSafe()
	if err != nil {
		return "", err
	}
	return pj.Environments, err
}

//Platforms returned are contrained and all variables evaluated
func (p *Project) Platforms() ([]projectfile.Platform, error) {
	pj, err := projectfile.GetSafe()
	if err != nil {
		return nil, err
	}
	return pj.Platforms, err
}

//Hooks returned are contrained and all variables evaluated
func (p *Project) Hooks() ([]Hook, error) {
	pj, err := projectfile.GetSafe()
	if err != nil {
		return nil, err
	}
	validHooks := make([]Hook, len(pj.Hooks))
	for _, hook := range pj.Hooks {
		if !constraints.IsConstrained(hook.Constraints) {
			newHook := Hook{}
			newHook.Name = hook.Name
			newHook.Value, err = variables.ExpandFromProject(hook.Value, pj)
			if err != nil {
				return nil, err
			}
			validHooks = append(validHooks, newHook)
		}
	}
	return validHooks, err
}

//Languages returned are contrained and all variables evaluated
func (p *Project) Languages() ([]Language, error) {
	pj, err := projectfile.GetSafe()
	if err != nil {
		return nil, err
	}
	validlangs := make([]Language, len(pj.Languages))
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

//Packages returned are contrained and all variables evaluated
func (l *Language) Packages() ([]Package, error) {
	validPackages := make([]Package, len(l.packages))
	for _, pkg := range l.packages {
		if !constraints.IsConstrained(pkg.Constraints) {
			newPkg := Package{}
			newPkg.Name = pkg.Name
			newPkg.Version = pkg.Version
			newPkg.Build = pkg.Build
			validPackages = append(validPackages, newPkg)
		}
	}
	return validPackages, nil
}

//Commands returned are contrained and all variables evaluated
func (p *Project) Commands() ([]Command, error) {
	pj, err := projectfile.GetSafe()
	if err != nil {
		return nil, err
	}
	validCmds := make([]Command, len(pj.Commands))
	for _, command := range pj.Commands {
		if !constraints.IsConstrained(command.Constraints) {
			newCmd := Command{}
			newCmd.Name = command.Name
			newCmd.Value = command.Value
			newCmd.Standalone = command.Standalone
			validCmds = append(validCmds, newCmd)
		}
	}
	return validCmds, err
}

//Variables returned are contrained and all variables evaluated
func (p *Project) Variables() ([]Variable, error) {
	pj, err := projectfile.GetSafe()
	if err != nil {
		return nil, err
	}
	validVars := make([]Variable, len(pj.Variables))
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
