package buildscript

import (
	"strings"

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

func (b *BuildScript) Requirements() ([]types.Requirement, error) {
	requirementsNode, err := b.getRequirementsNode()
	if err != nil {
		return nil, errs.Wrap(err, "Could not get requirements node")
	}

	var requirements []types.Requirement
	for _, req := range *requirementsNode.List {
		if req.FuncCall == nil {
			continue
		}

		var r types.Requirement
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
		requirements = append(requirements, r)
	}

	return requirements, nil
}

func (b *BuildScript) getRequirementsNode() (*Value, error) {
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

func getVersionRequirements(v *Value) []types.VersionRequirement {
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

func (b *BuildScript) getSolveNode() (*Value, error) {
	var search func([]*Assignment) *Value
	search = func(assignments []*Assignment) *Value {
		var nextLet []*Assignment
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

func (b *BuildScript) getSolveAtTimeValue() (*Value, error) {
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

func (b *BuildScript) getPlatformsNode() (*Value, error) {
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