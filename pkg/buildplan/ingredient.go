package buildplan

import (
	"github.com/ActiveState/cli/pkg/buildplan/raw"
	"github.com/go-openapi/strfmt"
)

type Ingredient struct {
	*raw.IngredientSource
	IsBuildtimeDependency bool
	IsRuntimeDependency   bool
	Platforms             []strfmt.UUID
	Artifacts             []*Artifact
}

type Ingredients []*Ingredient

type IngredientIDMap map[strfmt.UUID]*Ingredient

type IngredientNameMap map[string]*Ingredient

func (i Ingredients) ToIDMap() IngredientIDMap {
	result := make(map[strfmt.UUID]*Ingredient, len(i))
	for _, ig := range i {
		result[ig.IngredientID] = ig
	}
	return result
}

func (i Ingredients) ToNameMap() IngredientNameMap {
	result := make(map[string]*Ingredient, len(i))
	for _, ig := range i {
		result[ig.Name] = ig
	}
	return result
}

func (i *Ingredient) Dependencies(recursive bool) Ingredients {
	dependencies := Ingredients{}
	for _, a := range i.Artifacts {
		for _, ac := range a.children {
			dependencies = append(dependencies, ac.Ingredients...)
			if recursive {
				for _, ic := range ac.Ingredients {
					dependencies = append(dependencies, ic.Dependencies(recursive)...)
				}
			}
		}
	}
	return dependencies
}
