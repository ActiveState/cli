package project

import (
	"fmt"

	"github.com/ActiveState/cli/internal/constraints"

	"github.com/ActiveState/cli/pkg/projectfile"

	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/internal/variables"
)

// FailProjectNotLoaded identifies a failure as being due to a missing project file
var FailProjectNotLoaded = failures.Type("project.fail.notparsed", failures.FailUser)

//Hooks returned are contrained and all variables evaluated
func Hooks() ([]projectfile.Hook, error) {
	pj, err := projectfile.GetSafe()
	if err != nil {
		fmt.Println() //Do something
	}
	validHooks := make([]projectfile.Hook, len(pj.Hooks))
	for _, hook := range pj.Hooks {
		if !constraints.IsConstrained(hook.Constraints) {
			tmp := projectfile.Hook{}
			tmp.Name = hook.Name
			tmp.Value, err = variables.ExpandFromProject(hook.Value, pj)
			if err != nil {
				return nil, err
			}
			validHooks = append(validHooks, hook)
		}
	}
	return validHooks, err
}

//Languages returned are contrained and all variables evaluated
func Languages() ([]projectfile.Language, error) {
	pj, err := projectfile.GetSafe()
	if err != nil {
		return nil, err
	}
	validlangs := make([]projectfile.Language, len(pj.Languages))
	for _, language := range pj.Languages {
		if !constraints.IsConstrained(language.Constraints) {
			tmp := projectfile.Language{}
			tmp.Name = language.Name
			tmp.Version = language.Version
			tmp.Build = language.Build
			if language.Packages != nil {
				validPackages := make([]projectfile.Package, len(language.Packages))
				for _, pkg := range language.Packages {
					if !constraints.IsConstrained(pkg.Constraints) {
						validPackages = append(validPackages, pkg)
					}
				}
				tmp.Packages = validPackages
			}
			validlangs = append(validlangs, language)
		}
	}
	return validlangs, err
}

//PackagesOfLanguage returned are contrained and all variables evaluated
func PackagesOfLanguage(language projectfile.Language) []projectfile.Package {
	var validPackages []projectfile.Package
	if language.Packages != nil {
		validPackages := make([]projectfile.Package, len(language.Packages))
		for _, pkg := range language.Packages {
			if !constraints.IsConstrained(pkg.Constraints) {
				validPackages = append(validPackages, pkg)
			}
		}
	}
	return validPackages
}

//Commands returned are contrained and all variables evaluated
func Commands() ([]projectfile.Command, error) {
	pj, err := projectfile.GetSafe()
	if err != nil {
		return nil, err
	}
	validCmds := make([]projectfile.Command, len(pj.Commands))
	for _, command := range pj.Commands {
		if !constraints.IsConstrained(command.Constraints) {
			tmp := projectfile.Command{}
			tmp.Name = command.Name
			tmp.Value = command.Value
			tmp.Standalone = command.Standalone
			validCmds = append(validCmds, command)
		}
	}
	return validCmds, err
}

//Variables returned are contrained and all variables evaluated
func Variables() ([]projectfile.Variable, error) {
	pj, err := projectfile.GetSafe()
	if err != nil {
		return nil, err
	}
	validVars := make([]projectfile.Variable, len(pj.Variables))
	for _, variable := range pj.Variables {
		if !constraints.IsConstrained(variable.Constraints) {
			tmp := projectfile.Variable{}
			tmp.Name = variable.Name
			tmp.Value = variable.Value
			validVars = append(validVars, variable)
		}
	}
	return validVars, err
}
