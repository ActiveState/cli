package raw

import (
	"golang.org/x/text/cases"
	"golang.org/x/text/language"

	"github.com/ActiveState/cli/pkg/buildscript/internal/buildexpression"
)

func isLegacyRequirementsList(list *buildexpression.Var) bool {
	return len(*list.Value.List) > 0 && (*list.Value.List)[0].Object != nil
}

// transformRequirements transforms a buildexpression list of requirements in object form into a
// list of requirements in function-call form, which is how requirements are represented in
// buildscripts.
// This is to avoid custom marshaling code and reuse existing marshaling code.
func transformRequirements(reqs *buildexpression.Var) *buildexpression.Var {
	newReqs := &buildexpression.Var{
		Name: buildexpression.RequirementsKey,
		Value: &buildexpression.Value{
			List: &[]*buildexpression.Value{},
		},
	}

	for _, req := range *reqs.Value.List {
		*newReqs.Value.List = append(*newReqs.Value.List, transformRequirement(req))
	}

	return newReqs
}

// transformRequirement transforms a buildexpression requirement in object form into a requirement
// in function-call form.
// For example, transform something like
//
//	{"name": "<name>", "namespace": "<namespace>",
//		"version_requirements": [{"comparator": "<op>", "version": "<version>"}]}
//
// into something like
//
//	Req(name = "<name>", namespace = "<namespace>", version = <op>(value = "<version>"))
func transformRequirement(req *buildexpression.Value) *buildexpression.Value {
	newReq := &buildexpression.Value{
		Ap: &buildexpression.Ap{
			Name:      reqFuncName,
			Arguments: []*buildexpression.Value{},
		},
	}

	for _, arg := range *req.Object {
		name := arg.Name
		value := arg.Value

		// Transform the version value from the requirement object.
		if name == buildexpression.RequirementVersionRequirementsKey {
			name = buildexpression.RequirementVersionKey
			value = &buildexpression.Value{Ap: transformVersion(arg)}
		}

		// Add the argument to the function transformation.
		newReq.Ap.Arguments = append(newReq.Ap.Arguments, &buildexpression.Value{
			Assignment: &buildexpression.Var{Name: name, Value: value},
		})
	}

	return newReq
}

// transformVersion transforms a buildexpression version_requirements list in object form into
// function-call form.
// For example, transform something like
//
//	[{"comparator": "<op1>", "version": "<version1>"}, {"comparator": "<op2>", "version": "<version2>"}]
//
// into something like
//
//	And(<op1>(value = "<version1>"), <op2>(value = "<version2>"))
func transformVersion(requirements *buildexpression.Var) *buildexpression.Ap {
	var aps []*buildexpression.Ap
	for _, constraint := range *requirements.Value.List {
		ap := &buildexpression.Ap{}
		for _, o := range *constraint.Object {
			switch o.Name {
			case buildexpression.RequirementVersionKey:
				ap.Arguments = []*buildexpression.Value{{
					Assignment: &buildexpression.Var{Name: "value", Value: &buildexpression.Value{Str: o.Value.Str}},
				}}
			case buildexpression.RequirementComparatorKey:
				ap.Name = cases.Title(language.English).String(*o.Value.Str)
			}
		}
		aps = append(aps, ap)
	}

	if len(aps) == 1 {
		return aps[0] // e.g. Eq(value = "1.0")
	}

	// e.g. And(left = Gt(value = "1.0"), right = Lt(value = "3.0"))
	// Iterate backwards over the requirements array and construct a binary tree of 'And()' functions.
	// For example, given [Gt(value = "1.0"), Ne(value = "2.0"), Lt(value = "3.0")], produce:
	//   And(left = Gt(value = "1.0"), right = And(left = Ne(value = "2.0"), right = Lt(value = "3.0")))
	var ap *buildexpression.Ap
	for i := len(aps) - 2; i >= 0; i-- {
		right := &buildexpression.Value{Ap: aps[i+1]}
		if ap != nil {
			right = &buildexpression.Value{Ap: ap}
		}
		args := []*buildexpression.Value{
			{Assignment: &buildexpression.Var{Name: "left", Value: &buildexpression.Value{Ap: aps[i]}}},
			{Assignment: &buildexpression.Var{Name: "right", Value: right}},
		}
		ap = &buildexpression.Ap{Name: andFuncName, Arguments: args}
	}
	return ap
}
