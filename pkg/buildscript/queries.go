package buildscript

import (
	"strings"

	"github.com/ActiveState/cli/internal/logging"
	"github.com/go-openapi/strfmt"
	"github.com/thoas/go-funk"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/pkg/platform/api/buildplanner/types"
)

const (
	solveFuncName       = "solve"
	solveLegacyFuncName = "solve_legacy"
	srcKey              = "src"
	mergeKey            = "merge"
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

// Returns the requirements for the given target.
// If no target is given, uses the default target (i.e. the name assigned to 'main').
func (b *BuildScript) Requirements(targets ...string) ([]Requirement, error) {
	requirementsNode, err := b.getRequirementsNode(targets...)
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

// parseRequirement turns a raw *value representing a requirement into an externally consumable requirement type
// It accepts any value as input. If the value does not represent a requirement it simply won't be acted on and a nill
// will be returned.
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
				r.Name = *arg.Assignment.Value.Str
			case requirementNamespaceKey:
				r.Namespace = *arg.Assignment.Value.Str
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
				r.Name = *arg.Assignment.Value.Str
			case requirementRevisionIDKey:
				r.RevisionID = strfmt.UUID(*arg.Assignment.Value.Str)
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
func (b *BuildScript) DependencyRequirements(targets ...string) ([]types.Requirement, error) {
	reqs, err := b.Requirements(targets...)
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

func (b *BuildScript) getRequirementsNode(targets ...string) (*value, error) {
	node, err := b.getSolveNode(targets...)
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
			requirementVersionKey:    *v.FuncCall.Arguments[0].Assignment.Value.Str,
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

func isSolveFuncName(name string) bool {
	return name == solveFuncName || name == solveLegacyFuncName
}

func (b *BuildScript) getTargetSolveNode(targets ...string) (*value, error) {
	if len(targets) == 0 {
		for _, assignment := range b.raw.Assignments {
			if assignment.Key != mainKey {
				continue
			}
			if assignment.Value.Ident != nil && *assignment.Value.Ident != "" {
				targets = []string{*assignment.Value.Ident}
				break
			}
		}
	}

	var search func([]*assignment) *value
	search = func(assignments []*assignment) *value {
		var nextLet []*assignment
		for _, a := range assignments {
			if a.Key == letKey {
				nextLet = *a.Value.Object // nested 'let' to search next
				continue
			}

			if funk.Contains(targets, a.Key) && a.Value.FuncCall != nil {
				return a.Value
			}

			if f := a.Value.FuncCall; len(targets) == 0 && f != nil && isSolveFuncName(f.Name) {
				// This is coming from a complex build expression with no straightforward way to determine
				// a default target. Fall back on a top-level solve node.
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

func (b *BuildScript) getSolveNode(targets ...string) (*value, error) {
	node, err := b.getTargetSolveNode(targets...)
	if err != nil {
		return nil, errs.Wrap(err, "Could not get target node")
	}

	// If the target is the solve function, we're done.
	if isSolveFuncName(node.FuncCall.Name) {
		return node, nil
	}

	// If the target is a merge call, then look at right and left branches (in reverse order since the
	// right branch has precedence).
	if node.FuncCall.Name == mergeKey {
		for i := len(node.FuncCall.Arguments) - 1; i >= 0; i-- {
			arg := node.FuncCall.Arguments[i]
			if arg.Assignment == nil {
				continue
			}
			a := arg.Assignment
			if a.Value.Ident != nil {
				if node, err := b.getSolveNode(*a.Value.Ident); err == nil {
					return node, nil
				}
				// Note: ignore errors because either branch may not contain a solve node.
				// We'll return an error if both branches do not contain a solve node.
			}
		}
		return nil, errNodeNotFound
	}

	// Otherwise, the "src" key contains a reference to the solve node.
	// For example:
	//
	// runtime = state_tool_artifacts_v1(src = sources)
	// sources = solve(at_time = ..., platforms = [...], requirements = [...], ...)
	//
	// Look over the build expression again for that referenced node.
	for _, arg := range node.FuncCall.Arguments {
		if arg.Assignment == nil {
			continue
		}
		a := arg.Assignment
		if a.Key == srcKey && a.Value.Ident != nil {
			node, err := b.getSolveNode(*a.Value.Ident)
			if err != nil {
				return nil, errs.Wrap(err, "Could not get solve node from target")
			}
			return node, nil
		}
	}

	return nil, errNodeNotFound
}

func (b *BuildScript) getSolveAtTimeValue(targets ...string) (*value, error) {
	node, err := b.getSolveNode(targets...)
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

func (b *BuildScript) Platforms(targets ...string) ([]strfmt.UUID, error) {
	node, err := b.getPlatformsNode(targets...)
	if err != nil {
		return nil, errs.Wrap(err, "Could not get platform node")
	}

	list := []strfmt.UUID{}
	for _, value := range *node.List {
		list = append(list, strfmt.UUID(*value.Str))
	}
	return list, nil
}

func (b *BuildScript) getPlatformsNode(targets ...string) (*value, error) {
	node, err := b.getSolveNode(targets...)
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
