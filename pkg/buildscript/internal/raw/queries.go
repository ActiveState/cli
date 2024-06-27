package raw

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

func (r *Raw) Requirements() ([]types.Requirement, error) {
	requirementsNode, err := r.getRequirementsNode()
	if err != nil {
		return nil, errs.Wrap(err, "Could not get requirements node")
	}

	var requirements []types.Requirement
	for _, r := range *requirementsNode.List {
		if r.FuncCall == nil {
			continue
		}

		var req types.Requirement
		for _, arg := range r.FuncCall.Arguments {
			switch arg.Assignment.Key {
			case requirementNameKey:
				req.Name = strings.Trim(*arg.Assignment.Value.Str, `"`)
			case requirementNamespaceKey:
				req.Namespace = strings.Trim(*arg.Assignment.Value.Str, `"`)
			case requirementVersionKey:
				req.VersionRequirement = getVersionRequirements(arg.Assignment.Value)
			}
		}
		requirements = append(requirements, req)
	}

	return requirements, nil
}

func (r *Raw) getRequirementsNode() (*Value, error) {
	node, err := r.getSolveNode()
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
			requirementVersionKey:    strings.Trim(*v.FuncCall.Arguments[0].Assignment.Value.Str, `"`),
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

func (r *Raw) getSolveNode() (*Value, error) {
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
	if node := search(r.Assignments); node != nil {
		return node, nil
	}

	return nil, errNodeNotFound
}

func (r *Raw) getSolveAtTimeValue() (*Value, error) {
	node, err := r.getSolveNode()
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

func (r *Raw) Platforms() ([]strfmt.UUID, error) {
	node, err := r.getPlatformsNode()
	if err != nil {
		return nil, errs.Wrap(err, "Could not get platform node")
	}

	list := []strfmt.UUID{}
	for _, value := range *node.List {
		list = append(list, strfmt.UUID(strings.Trim(*value.Str, `"`)))
	}
	return list, nil
}

func (r *Raw) getPlatformsNode() (*Value, error) {
	node, err := r.getSolveNode()
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
