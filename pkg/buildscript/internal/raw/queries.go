package raw

import (
	"strings"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/pkg/platform/api/buildplanner/types"
)

const (
	solveFuncName       = "solve"
	solveLegacyFuncName = "solve_legacy"
	platformsKey        = "platforms"
	letKey              = "let"
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
			case RequirementNameKey:
				req.Name = strings.Trim(*arg.Assignment.Value.Str, `"`)
			case RequirementNamespaceKey:
				req.Namespace = strings.Trim(*arg.Assignment.Value.Str, `"`)
			case RequirementVersionKey:
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
	case eqFuncName, neFuncName, gtFuncName, gteFuncName, ltFuncName, lteFuncName:
		reqs = append(reqs, types.VersionRequirement{
			RequirementComparatorKey: strings.ToLower(v.FuncCall.Name),
			RequirementVersionKey:    strings.Trim(*v.FuncCall.Arguments[0].Assignment.Value.Str, `"`),
		})

	case andFuncName:
		for _, arg := range v.FuncCall.Arguments {
			if arg.Assignment != nil && arg.Assignment.Value.FuncCall != nil {
				reqs = append(reqs, getVersionRequirements(arg.Assignment.Value)...)
			}
		}
	}

	return reqs
}

// getSolveNode returns the solve node from the build expression.
// It returns an error if the solve node is not found.
// Currently, the solve node can have the name of "solve" or "solve_legacy".
// It expects the JSON representation of the build expression to be formatted as follows:
//
//	{
//	  "let": {
//	    "runtime": {
//	      "solve": {
//	      }
//	    }
//	  }
//	}
func (r *Raw) getSolveNode() (*Value, error) {
	var search func([]*Assignment) *Value
	search = func(assignments []*Assignment) *Value {
		var nextLet []*Assignment
		for _, a := range assignments {
			if a.Key == letKey {
				nextLet = *a.Value.Object // nested 'let' to search next
				continue
			}

			if a.Value.FuncCall == nil {
				continue
			}

			if a.Value.FuncCall.Name == solveFuncName || a.Value.FuncCall.Name == solveLegacyFuncName {
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

func (r *Raw) getSolveNodeArguments() ([]*Value, error) {
	node, err := r.getSolveNode()
	if err != nil {
		return nil, errs.Wrap(err, "Could not get solve node")
	}

	return node.FuncCall.Arguments, nil
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

func (r *Raw) getPlatformsNode() (*[]*Value, error) {
	node, err := r.getSolveNode()
	if err != nil {
		return nil, errs.Wrap(err, "Could not get solve node")
	}

	for _, arg := range node.FuncCall.Arguments {
		if arg.Assignment == nil {
			continue
		}

		if arg.Assignment.Key == platformsKey && arg.Assignment.Value != nil {
			return arg.Assignment.Value.List, nil
		}
	}

	return nil, errNodeNotFound
}
