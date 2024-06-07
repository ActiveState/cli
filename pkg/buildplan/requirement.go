package buildplan

import "github.com/ActiveState/cli/pkg/platform/api/buildplanner/types"

type Requirements []*Requirement

type Requirement struct {
	*types.Requirement
	// Requirements are not ingredients, because multiple requirements can be satisfied by the same ingredient,
	// for example `rake` is satisfied by `ruby`.
	Ingredient *Ingredient
}
