package buildplan

import "github.com/go-openapi/strfmt"

type Ingredient struct {
	*Source
	IsBuildtimeDependency bool
	IsRuntimeDependency   bool
	Platforms             []strfmt.UUID
	Artifacts             []*Artifact
	Children              []*Ingredient
	Parents               []*Ingredient
}

type Ingredients []*Ingredient

func (i Ingredients) ToMap() map[strfmt.UUID]*Ingredient {
	result := make(map[strfmt.UUID]*Ingredient, len(i))
	for _, ig := range i {
		result[ig.NodeID] = ig
	}
	return result
}
