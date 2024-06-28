package buildscript

import (
	"strings"

	"github.com/go-openapi/strfmt"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/pkg/ascript"
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
				r.Name = ascript.StrValue(arg.Assignment.Value)
			case requirementNamespaceKey:
				r.Namespace = ascript.StrValue(arg.Assignment.Value)
			case requirementVersionKey:
				r.VersionRequirement = getVersionRequirements(arg.Assignment.Value)
			}
		}
		requirements = append(requirements, r)
	}

	return requirements, nil
}

func (b *BuildScript) getRequirementsNode() (*ascript.Value, error) {
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

func getVersionRequirements(v *ascript.Value) []types.VersionRequirement {
	reqs := []types.VersionRequirement{}

	switch v.FuncCall.Name {
	// e.g. Eq(value = "1.0")
	case ascript.EqFuncName, ascript.NeFuncName, ascript.GtFuncName, ascript.GteFuncName, ascript.LtFuncName, ascript.LteFuncName:
		reqs = append(reqs, types.VersionRequirement{
			requirementComparatorKey: strings.ToLower(v.FuncCall.Name),
			requirementVersionKey:    ascript.StrValue(v.FuncCall.Arguments[0].Assignment.Value),
		})

	// e.g. And(left = Gte(value = "1.0"), right = Lt(value = "2.0"))
	case ascript.AndFuncName:
		for _, arg := range v.FuncCall.Arguments {
			if arg.Assignment != nil && arg.Assignment.Value.FuncCall != nil {
				reqs = append(reqs, getVersionRequirements(arg.Assignment.Value)...)
			}
		}
	}

	return reqs
}

func (b *BuildScript) getSolveNode() (*ascript.Value, error) {
	var search func([]*ascript.Assignment) *ascript.Value
	search = func(assignments []*ascript.Assignment) *ascript.Value {
		var nextLet []*ascript.Assignment
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
	if node := search(b.as.Assignments); node != nil {
		return node, nil
	}

	return nil, errNodeNotFound
}

func (b *BuildScript) getSolveAtTimeValue() (*ascript.Value, error) {
	node, err := b.getSolveNode()
	if err != nil {
		return nil, errs.Wrap(err, "Could not get solve node")
	}

	for _, arg := range node.FuncCall.Arguments {
		if arg.Assignment != nil && arg.Assignment.Key == ascript.AtTimeKey {
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
		list = append(list, strfmt.UUID(ascript.StrValue(value)))
	}
	return list, nil
}

func (b *BuildScript) getPlatformsNode() (*ascript.Value, error) {
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
