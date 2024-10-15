package buildscript

import (
	"strings"

	"github.com/ActiveState/cli/internal/logging"
	"github.com/go-openapi/strfmt"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/pkg/platform/api/buildplanner/types"
)

const (
	solveFuncName       = "solve"
	solveLegacyFuncName = "solve_legacy"
	requirementsKey     = "requirements"
	platformsKey        = "platforms"
)

var errNodeNotFound = errs.New("Could not find node")
var errValueNotFound = errs.New("Could not find value")

type Requirement interface {
	IsRequirement()
}

type DependencyRequirement struct {
	types.Requirement
}

func (r DependencyRequirement) IsRequirement() {}

type RevisionRequirement struct {
	Name       string      `json:"name"`
	RevisionID strfmt.UUID `json:"revision_id"`
}

func (r RevisionRequirement) IsRequirement() {}

type UnknownRequirement struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

func (r UnknownRequirement) IsRequirement() {}

func (b *BuildScript) Requirements() ([]Requirement, error) {
	requirementsNode, err := b.getRequirementsNode()
	if err != nil {
		return nil, errs.Wrap(err, "Could not get requirements node")
	}

	return exportRequirements(requirementsNode), nil
}

func exportRequirements(v *value) []Requirement {
	if v.List == nil {
		logging.Error("exportRequirements called with value that does not have a list")
		return nil
	}
	var requirements []Requirement
	for _, req := range *v.List {
		if req.FuncCall == nil {
			continue
		}

		switch req.FuncCall.Name {
		case reqFuncName, revFuncName:
			requirements = append(requirements, parseRequirement(req))
		default:
			requirements = append(requirements, UnknownRequirement{
				Name:  req.FuncCall.Name,
				Value: argsToString(req.FuncCall.Arguments, "", ", ", func(v string) string { return v }),
			})
		}

	}

	return requirements
}

func parseRequirement(req *value) Requirement {
	if req.FuncCall == nil {
		return nil
	}
	switch req.FuncCall.Name {
	case reqFuncName:
		var r DependencyRequirement
		for _, arg := range req.FuncCall.Arguments {
			switch arg.Assignment.Key {
			case requirementNameKey:
				r.Name = strValue(arg.Assignment.Value)
			case requirementNamespaceKey:
				r.Namespace = strValue(arg.Assignment.Value)
			case requirementVersionKey:
				r.VersionRequirement = getVersionRequirements(arg.Assignment.Value)
			}
		}
		return r
	case revFuncName:
		var r RevisionRequirement
		for _, arg := range req.FuncCall.Arguments {
			switch arg.Assignment.Key {
			case requirementNameKey:
				r.Name = strValue(arg.Assignment.Value)
			case requirementRevisionIDKey:
				r.RevisionID = strfmt.UUID(strValue(arg.Assignment.Value))
			}
		}
		return r
	default:
		return nil
	}
}

// DependencyRequirements is identical to Requirements except that it only considers dependency type requirements,
// which are the most common.
// ONLY use this when you know you only need to care about dependencies.
func (b *BuildScript) DependencyRequirements() ([]types.Requirement, error) {
	reqs, err := b.Requirements()
	if err != nil {
		return nil, errs.Wrap(err, "Could not get requirements")
	}
	var deps []types.Requirement
	for _, req := range reqs {
		if dep, ok := req.(DependencyRequirement); ok {
			deps = append(deps, dep.Requirement)
		}
	}
	return deps, nil
}

func (b *BuildScript) getRequirementsNode() (*value, error) {
	node, err := b.getSolveNode()
	if err != nil {
		return nil, errs.Wrap(err, "Could not get solve node")
	}

	for _, arg := range node.FuncCall.Arguments {
		if arg.Assignment != nil && arg.Assignment.Key == requirementsKey {
			return arg.Assignment.Value, nil
		}
	}

	return nil, errNodeNotFound
}

func getVersionRequirements(v *value) []types.VersionRequirement {
	reqs := []types.VersionRequirement{}

	switch v.FuncCall.Name {
	// e.g. Eq(value = "1.0")
	case eqFuncName, neFuncName, gtFuncName, gteFuncName, ltFuncName, lteFuncName:
		reqs = append(reqs, types.VersionRequirement{
			requirementComparatorKey: strings.ToLower(v.FuncCall.Name),
			requirementVersionKey:    strValue(v.FuncCall.Arguments[0].Assignment.Value),
		})

	// e.g. And(left = Gte(value = "1.0"), right = Lt(value = "2.0"))
	case andFuncName:
		for _, arg := range v.FuncCall.Arguments {
			if arg.Assignment != nil && arg.Assignment.Value.FuncCall != nil {
				reqs = append(reqs, getVersionRequirements(arg.Assignment.Value)...)
			}
		}
	}

	return reqs
}

func (b *BuildScript) getSolveNode() (*value, error) {
	var search func([]*assignment) *value
	search = func(assignments []*assignment) *value {
		var nextLet []*assignment
		for _, a := range assignments {
			if a.Key == letKey {
				nextLet = *a.Value.Object // nested 'let' to search next
				continue
			}

			if f := a.Value.FuncCall; f != nil && (f.Name == solveFuncName || f.Name == solveLegacyFuncName) {
				return a.Value
			}
		}

		// The highest level solve node is not found, so recurse into the next let.
		if nextLet != nil {
			return search(nextLet)
		}

		return nil
	}
	if node := search(b.raw.Assignments); node != nil {
		return node, nil
	}

	return nil, errNodeNotFound
}

func (b *BuildScript) getSolveAtTimeValue() (*value, error) {
	node, err := b.getSolveNode()
	if err != nil {
		return nil, errs.Wrap(err, "Could not get solve node")
	}

	for _, arg := range node.FuncCall.Arguments {
		if arg.Assignment != nil && arg.Assignment.Key == atTimeKey {
			return arg.Assignment.Value, nil
		}
	}

	return nil, errValueNotFound
}

func (b *BuildScript) Platforms() ([]strfmt.UUID, error) {
	node, err := b.getPlatformsNode()
	if err != nil {
		return nil, errs.Wrap(err, "Could not get platform node")
	}

	list := []strfmt.UUID{}
	for _, value := range *node.List {
		list = append(list, strfmt.UUID(strValue(value)))
	}
	return list, nil
}

func (b *BuildScript) getPlatformsNode() (*value, error) {
	node, err := b.getSolveNode()
	if err != nil {
		return nil, errs.Wrap(err, "Could not get solve node")
	}

	for _, arg := range node.FuncCall.Arguments {
		if arg.Assignment != nil && arg.Assignment.Key == platformsKey {
			return arg.Assignment.Value, nil
		}
	}

	return nil, errNodeNotFound
}
