package raw

import (
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/pkg/platform/api/buildplanner/types"
)

const (
	solveFuncName       = "solve"
	solveLegacyFuncName = "solve_legacy"
	platformsKey        = "platforms"
)

var funcNodeNotFoundError = errs.New("Could not find function node")

func (r *Raw) Requirements() ([]types.Requirement, error) {
	requirementsNode, err := r.getRequirementsNode()
	if err != nil {
		return nil, errs.Wrap(err, "Could not get requirements node")
	}

	var requirements []types.Requirement
	for _, r := range requirementsNode {
		if r.Object == nil {
			continue
		}

		var req types.Requirement
		for _, o := range *r.Object {
			if o.Key == RequirementNameKey {
				req.Name = *o.Value.Str
			}

			if o.Key == RequirementNamespaceKey {
				req.Namespace = *o.Value.Str
			}

			if o.Key == RequirementVersionRequirementsKey {
				req.VersionRequirement = getVersionRequirements(o.Value.List)
			}
		}
		requirements = append(requirements, req)
	}

	return requirements, nil
}

func (r *Raw) getRequirementsNode() ([]*Value, error) {
	solveFunc, err := r.getSolveNode()
	if err != nil {
		return nil, errs.Wrap(err, "Could not get solve node")
	}

	var reqs []*Value
	for _, arg := range solveFunc.Arguments {
		if arg.Assignment == nil {
			continue
		}

		if arg.Assignment.Key == requirementsKey && arg.Assignment.Value != nil {
			reqs = *arg.Assignment.Value.List
		}
	}

	return reqs, nil
}

func getVersionRequirements(v *[]*Value) []types.VersionRequirement {
	var reqs []types.VersionRequirement

	if v == nil {
		return reqs
	}

	for _, r := range *v {
		if r.Object == nil {
			continue
		}

		versionReq := make(types.VersionRequirement)
		for _, o := range *r.Object {
			if o.Key == RequirementComparatorKey {
				versionReq[RequirementComparatorKey] = *o.Value.Str
			}

			if o.Key == RequirementVersionKey {
				versionReq[RequirementVersionKey] = *o.Value.Str
			}
		}
		reqs = append(reqs, versionReq)
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
func (r *Raw) getSolveNode() (*FuncCall, error) {
	// Search for solve node in the top level assignments.
	for _, a := range r.Assignments {
		if a.Value.FuncCall == nil {
			continue
		}

		if a.Value.FuncCall.Name == solveFuncName || a.Value.FuncCall.Name == solveLegacyFuncName {
			return a.Value.FuncCall, nil
		}
	}

	return nil, funcNodeNotFoundError
}

func (r *Raw) getSolveNodeArguments() ([]*Value, error) {
	solveFunc, err := r.getSolveNode()
	if err != nil {
		return nil, errs.Wrap(err, "Could not get solve node")
	}

	return solveFunc.Arguments, nil
}

func (r *Raw) getSolveAtTimeValue() (*Value, error) {
	solveFunc, err := r.getSolveNode()
	if err != nil {
		return nil, errs.Wrap(err, "Could not get solve node")
	}

	for _, arg := range solveFunc.Arguments {
		if arg.Assignment != nil && arg.Assignment.Key == atTimeKey {
			return arg.Assignment.Value, nil
		}
	}

	return nil, errs.New("Could not find %s", atTimeKey)
}

func (r *Raw) getPlatformsNode() (*[]*Value, error) {
	solveFunc, err := r.getSolveNode()
	if err != nil {
		return nil, errs.Wrap(err, "Could not get solve node")
	}

	for _, arg := range solveFunc.Arguments {
		if arg.Assignment == nil {
			continue
		}

		if arg.Assignment.Key == platformsKey && arg.Assignment.Value != nil {
			return arg.Assignment.Value.List, nil
		}
	}

	return nil, errs.New("Could not find platforms node")
}
