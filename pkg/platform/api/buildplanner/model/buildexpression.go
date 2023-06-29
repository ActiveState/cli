package model

import (
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/rtutils/p"
	"github.com/go-openapi/strfmt"
)

type Operation int

const (
	ComparatorEQ  string = "eq"
	ComparatorGT         = "gt"
	ComparatorGTE        = "gte"
	ComparatorLT         = "lt"
	ComparatorLTE        = "lte"
	ComparatorNE         = "ne"

	OperationAdded Operation = iota
	OperationRemoved
	OperationUpdated

	SolveFuncName                     = "solve"
	SolveLegacyFuncName               = "solve_legacy"
	RequirementsKey                   = "requirements"
	AtTimeKey                         = "at_time"
	RequirementNameKey                = "name"
	RequirementNamespaceKey           = "namespace"
	RequirementVersionRequirementsKey = "version_requirements"
	RequirementVersionKey             = "version"
	RequirementComparatorKey          = "comparator"

	ctxLet         = "let"
	ctxIn          = "in"
	ctxAp          = "ap"
	ctxValue       = "value"
	ctxAssignments = "assignments"
	ctxIsAp        = "isAp"
)

func (o Operation) String() string {
	switch o {
	case OperationAdded:
		return "added"
	case OperationRemoved:
		return "removed"
	case OperationUpdated:
		return "updated"
	default:
		return "unknown"
	}
}

var funcNodeNotFoundError = errors.New("Could not find function node")

type Requirement struct {
	Name               string               `json:"name"`
	Namespace          string               `json:"namespace"`
	VersionRequirement []VersionRequirement `json:"version_requirements,omitempty"`
}

type VersionRequirement map[string]string

type BuildExpression struct {
	Let *Let
}

type Let struct {
	Assignments []*Var
	In          *In
}

type Var struct {
	Name  string
	Value *Value
}

type Value struct {
	Ap   *Ap
	List *[]*Value
	Str  *string
	Null *Null

	Assignment *Var
	Object     *[]*Var
	Ident      *string
}

type Null struct {
	Null string
}

type Ap struct {
	Name      string
	Arguments []*Value
}

type In struct {
	FuncCall *Ap
	Name     *string
}

type context []string

func (s *context) push(str string) {
	*s = append(*s, str)
}

func (s *context) pop() (string, error) {
	if len(*s) == 0 {
		return "", errs.New("stack is empty")
	}
	str := (*s)[len(*s)-1]
	*s = (*s)[:len(*s)-1]
	return str, nil
}

func (s *context) contains(str string) bool {
	for _, v := range *s {
		if v == str {
			return true
		}
	}
	return false
}

// NewBuildExpression creates a BuildExpression from a JSON byte array.
// The JSON must be a valid BuildExpression in the following format:
//
//	{
//	  "let": {
//	    "runtime": {
//	      "solve_legacy": {
//	        "at_time": "2023-04-27T17:30:05.999000Z",
//	        "build_flags": [],
//	        "camel_flags": [],
//	        "platforms": [
//	          "96b7e6f2-bebf-564c-bc1c-f04482398f38"
//	        ],
//	        "requirements": [
//	          {
//	            "name": "requests",
//	            "namespace": "language/python"
//	          },
//	          {
//	            "name": "python",
//	            "namespace": "language",
//	            "version_requirements": [
//	              {
//	                "comparator": "eq",
//	                "version": "3.10.10"
//	              }
//	            ]
//	          },
//	        ],
//	        "solver_version": null
//	      }
//	    },
//	  "in": "$runtime"
//	  }
//	}
func NewBuildExpression(data []byte) (*BuildExpression, error) {
	m := make(map[string]interface{})
	err := json.Unmarshal(data, &m)
	if err != nil {
		return nil, errs.Wrap(err, "Could not unmarshal build expression")
	}

	letValue, ok := m["let"]
	if !ok {
		return nil, errs.New("Build expression has no 'let' key")
	}
	letMap, ok := letValue.(map[string]interface{})
	if !ok {
		return nil, errs.New("'let' key is not a JSON object")
	}

	ctx := context{}
	let, err := newLet(ctx, letMap)
	if err != nil {
		return nil, errs.Wrap(err, "Could not parse 'let' key")
	}

	expr := &BuildExpression{let}

	err = expr.validateRequirements()
	if err != nil {
		return nil, errs.Wrap(err, "Could not validate requirements")
	}

	return expr, nil
}

func newLet(ctx context, m map[string]interface{}) (*Let, error) {
	ctx.push(ctxLet)
	defer ctx.pop()

	inValue, ok := m["in"]
	if !ok {
		return nil, errs.New("Build expression's 'let' object has no 'in' key")
	}

	in, err := newIn(ctx, inValue)
	if err != nil {
		return nil, errs.Wrap(err, "Could not parse 'in' key's value: %v", inValue)
	}

	// Delete in so it doesn't get parsed as an assignment.
	delete(m, "in")

	assignments, err := newAssignments(ctx, m)
	if err != nil {
		return nil, errs.Wrap(err, "Could not parse 'let' key")
	}

	return &Let{Assignments: *assignments, In: in}, nil
}

func isAp(ctx context, value map[string]interface{}) bool {
	ctx.push(ctxIsAp)
	defer ctx.pop()

	_, hasIn := value["in"]
	if hasIn && !ctx.contains(ctxAssignments) {
		return false
	}

	return true
}

func newValue(ctx context, valueInterface interface{}) (*Value, error) {
	ctx.push(ctxValue)
	defer ctx.pop()

	value := &Value{}

	switch v := valueInterface.(type) {
	case map[string]interface{}:
		// Examine keys first to see if this is a function call.
		for key, val := range v {
			if _, ok := val.(map[string]interface{}); !ok {
				continue
			}

			if isAp(ctx, val.(map[string]interface{})) {
				f, err := newAp(ctx, v)
				if err != nil {
					return nil, errs.Wrap(err, "Could not parse '%s' function's value: %v", key, v)
				}
				value.Ap = f
			}
		}

		if value.Ap == nil {
			// It's not a function call, but an object.
			object, err := newAssignments(ctx, v)
			if err != nil {
				return nil, errs.Wrap(err, "Could not parse object: %v", v)
			}
			value.Object = object
		}

	case []interface{}:
		values := []*Value{}
		for _, item := range v {
			value, err := newValue(ctx, item)
			if err != nil {
				return nil, errs.Wrap(err, "Could not parse list: %v", v)
			}
			values = append(values, value)
		}
		value.List = &values

	case string:
		if ctx.contains(ctxIn) {
			value.Ident = &v
		} else {
			value.Str = p.StrP(v)
		}

	default:
		// An empty value is interpreted as JSON null.
		value.Null = &Null{}
	}

	return value, nil
}

func newAp(ctx context, m map[string]interface{}) (*Ap, error) {
	ctx.push(ctxAp)
	defer ctx.pop()

	// Look in the given object for the function's name and argument object or list.
	var name string
	var argsInterface interface{}
	for key, value := range m {
		if isAp(ctx, value.(map[string]interface{})) {
			name = key
			argsInterface = value
			break
		}
	}

	args := []*Value{}

	switch v := argsInterface.(type) {
	case map[string]interface{}:
		for key, valueInterface := range v {
			value, err := newValue(ctx, valueInterface)
			if err != nil {
				return nil, errs.Wrap(err, "Could not parse '%s' function's argument '%s': %v", name, key, valueInterface)
			}
			args = append(args, &Value{Assignment: &Var{Name: key, Value: value}})
		}
		sort.SliceStable(args, func(i, j int) bool { return args[i].Assignment.Name < args[j].Assignment.Name })

	case []interface{}:
		for _, item := range v {
			value, err := newValue(ctx, item)
			if err != nil {
				return nil, errs.Wrap(err, "Could not parse '%s' function's argument list item: %v", name, item)
			}
			args = append(args, value)
		}

	default:
		return nil, errs.New("Function '%s' expected to be object or list", name)
	}

	return &Ap{Name: name, Arguments: args}, nil
}

func newAssignments(ctx context, m map[string]interface{}) (*[]*Var, error) {
	ctx.push(ctxAssignments)
	defer ctx.pop()

	assignments := []*Var{}
	for key, valueInterface := range m {
		value, err := newValue(ctx, valueInterface)
		if err != nil {
			return nil, errs.Wrap(err, "Could not parse '%s' key's value: %v", key, valueInterface)
		}
		assignments = append(assignments, &Var{Name: key, Value: value})
	}
	sort.SliceStable(assignments, func(i, j int) bool { return assignments[i].Name < assignments[j].Name })
	return &assignments, nil
}

func newIn(ctx context, inValue interface{}) (*In, error) {
	ctx.push(ctxIn)
	defer ctx.pop()

	in := &In{}

	switch v := inValue.(type) {
	case map[string]interface{}:
		f, err := newAp(ctx, v)
		if err != nil {
			return nil, errs.Wrap(err, "'in' object is not a function call")
		}
		in.FuncCall = f

	case string:
		in.Name = p.StrP(strings.TrimPrefix(v, "$"))

	default:
		return nil, errs.New("'in' value expected to be a function call or string")
	}

	return in, nil
}

// validateRequirements ensures that the requirements in the BuildExpression contain
// both the name and namespace fields. These fileds are used for requirement operations.
func (e *BuildExpression) validateRequirements() error {
	requirements := e.getRequirementsNode()
	for _, r := range requirements {
		if r.Object == nil {
			continue
		}

		// The requirement object needs to have a name and value field.
		// The value can be a string (in the case of name or namespace)
		// or a list (in the case of version requirements).
		for _, o := range *r.Object {
			if o.Name == "" {
				return errs.New("Requirement object missing name field")
			}

			if o.Value == nil {
				return errs.New("Requirement object missing value field")
			}

			if o.Value.Str == nil && o.Value.List == nil {
				return errs.New("Requirement object value field is not a string")
			}
		}
	}
	return nil
}

// Requirements returns the requirements in the BuildExpression.
// It returns an error if the requirements are not found or if they are malformed.
// It expects the JSON representation of the solve node to be formatted as follows:
//
//	{
//	  "requirements": [
//	    {
//	      "name": "requests",
//	      "namespace": "language/python"
//	    },
//	    {
//	      "name": "python",
//	      "namespace": "language",
//	      "version_requirements": [{
//	          "comparator": "eq",
//	          "version": "3.10.10"
//	      }]
//	    }
//	  ]
//	}
func (e *BuildExpression) Requirements() []Requirement {
	requirementsNode := e.getRequirementsNode()

	var requirements []Requirement
	for _, r := range requirementsNode {
		if r.Object == nil {
			continue
		}

		var req Requirement
		for _, o := range *r.Object {
			if o.Name == RequirementNameKey {
				req.Name = *o.Value.Str
			}

			if o.Name == RequirementNamespaceKey {
				req.Namespace = *o.Value.Str
			}

			if o.Name == RequirementVersionRequirementsKey {
				req.VersionRequirement = getVersionRequirements(o.Value.List)
			}
		}
		requirements = append(requirements, req)
	}

	return requirements
}

func (e *BuildExpression) getRequirementsNode() []*Value {
	solveAp := e.getSolveNode()

	var reqs []*Value
	for _, arg := range solveAp.Arguments {
		if arg.Assignment == nil {
			continue
		}

		if arg.Assignment.Name == RequirementsKey && arg.Assignment.Value != nil {
			reqs = *arg.Assignment.Value.List
		}
	}

	return reqs
}

func getVersionRequirements(v *[]*Value) []VersionRequirement {
	var reqs []VersionRequirement
	for _, r := range *v {
		if r.Object == nil {
			continue
		}

		versionReq := make(VersionRequirement)
		for _, o := range *r.Object {
			if o.Name == RequirementComparatorKey {
				versionReq[RequirementComparatorKey] = *o.Value.Str
			}

			if o.Name == RequirementVersionKey {
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
func (e *BuildExpression) getSolveNode() *Ap {
	for _, a := range e.Let.Assignments {
		if a.Value.Ap == nil {
			continue
		}

		if a.Value.Ap.Name == SolveFuncName || a.Value.Ap.Name == SolveLegacyFuncName {
			return a.Value.Ap
		}
	}

	return nil
}

// Update updates the BuildExpression's requirements based on the operation and requirement.
func (e *BuildExpression) Update(operation Operation, requirement Requirement, timestamp strfmt.DateTime) error {
	var err error
	switch operation {
	case OperationAdded:
		err = e.addRequirement(requirement)
	case OperationRemoved:
		err = e.removeRequirement(requirement)
	case OperationUpdated:
		err = e.updateRequirement(requirement)
	default:
		return errs.New("Unsupported operation")
	}
	if err != nil {
		return errs.Wrap(err, "Could not update BuildExpression's requirements")
	}

	err = e.updateTimestamp(timestamp)
	if err != nil {
		return errs.Wrap(err, "Could not update BuildExpression's timestamp")
	}

	return nil
}

func (e *BuildExpression) addRequirement(requirement Requirement) error {
	obj := []*Var{
		{Name: RequirementNameKey, Value: &Value{Str: p.StrP(requirement.Name)}},
		{Name: RequirementNamespaceKey, Value: &Value{Str: p.StrP(requirement.Namespace)}},
	}

	if requirement.VersionRequirement != nil {
		for _, r := range requirement.VersionRequirement {
			obj = append(obj, &Var{Name: RequirementVersionRequirementsKey, Value: &Value{List: &[]*Value{
				{Object: &[]*Var{
					{Name: RequirementComparatorKey, Value: &Value{Str: p.StrP(r[RequirementComparatorKey])}},
					{Name: RequirementVersionKey, Value: &Value{Str: p.StrP(r[RequirementVersionKey])}},
				}}},
			}})
		}
	}

	requirementsNode := append(e.getRequirementsNode(), &Value{Object: &obj})

	for _, arg := range e.getSolveNode().Arguments {
		if arg.Assignment == nil {
			continue
		}

		if arg.Assignment.Name == RequirementsKey {
			arg.Assignment.Value.List = &requirementsNode
		}
	}

	return nil
}

func (e *BuildExpression) removeRequirement(requirement Requirement) error {
	requirementsNode := e.getRequirementsNode()

	for i, r := range requirementsNode {
		if r.Object == nil {
			continue
		}

		for _, o := range *r.Object {
			if o.Name == RequirementNameKey && *o.Value.Str == requirement.Name {
				requirementsNode = append(requirementsNode[:i], requirementsNode[i+1:]...)
			}
		}
	}

	for _, arg := range e.getSolveNode().Arguments {
		if arg.Assignment == nil {
			continue
		}

		if arg.Assignment.Name == RequirementsKey {
			arg.Assignment.Value.List = &requirementsNode
		}
	}

	return nil
}

func (e *BuildExpression) updateRequirement(requirement Requirement) error {
	requirementsNode := e.getRequirementsNode()

	for _, r := range requirementsNode {
		if r.Object == nil {
			continue
		}

		for _, o := range *r.Object {
			if o.Name == RequirementNameKey && *o.Value.Str == requirement.Name {
				if requirement.VersionRequirement != nil {
					for _, v := range *r.Object {
						if v.Name == "version_requirements" {
							v.Value.List = &[]*Value{
								{Object: &[]*Var{
									{Name: "comparator", Value: &Value{Str: p.StrP(requirement.VersionRequirement[0]["comparator"])}},
									{Name: "version", Value: &Value{Str: p.StrP(requirement.VersionRequirement[0]["version"])}},
								}},
							}
						}
					}
				}
			}
		}
	}

	for _, arg := range e.getSolveNode().Arguments {
		if arg.Assignment == nil {
			continue
		}

		if arg.Assignment.Name == RequirementsKey {
			arg.Assignment.Value.List = &requirementsNode
		}
	}

	return nil
}

func (e *BuildExpression) updateTimestamp(timestamp strfmt.DateTime) error {
	formatted, err := time.Parse(time.RFC3339, timestamp.String())
	if err != nil {
		return errs.Wrap(err, "Could not parse latest timestamp")
	}

	for _, arg := range e.getSolveNode().Arguments {
		if arg.Assignment == nil {
			continue
		}

		if arg.Assignment.Name == "at_time" {
			arg.Assignment.Value.Str = p.StrP(formatted.Format(time.RFC3339))
		}
	}

	return nil
}

func (e *BuildExpression) MarshalJSON() ([]byte, error) {
	m := make(map[string]interface{})
	let := make(map[string]interface{})
	for _, assignment := range e.Let.Assignments {
		let[assignment.Name] = assignment.Value
	}
	m["let"] = let
	return json.Marshal(m)
}

func (a *Var) MarshalJSON() ([]byte, error) {
	m := make(map[string]interface{})
	m[a.Name] = a.Value
	return json.Marshal(m)
}

func (v *Value) MarshalJSON() ([]byte, error) {
	switch {
	case v.Ap != nil:
		return json.Marshal(v.Ap)
	case v.List != nil:
		return json.Marshal(v.List)
	case v.Str != nil:
		return json.Marshal(strings.Trim(*v.Str, `"`))
	case v.Null != nil:
		return json.Marshal(nil)
	case v.Assignment != nil:
		return json.Marshal(v.Assignment)
	case v.Object != nil:
		m := make(map[string]interface{})
		for _, assignment := range *v.Object {
			m[assignment.Name] = assignment.Value
		}
		return json.Marshal(m)
	case v.Ident != nil:
		return json.Marshal(v.Ident)
	}
	return json.Marshal([]*Value{}) // participle does not create v.List if it's empty
}

func (f *Ap) MarshalJSON() ([]byte, error) {
	m := make(map[string]interface{})
	args := make(map[string]interface{})
	for _, argument := range f.Arguments {
		switch {
		case argument.Assignment != nil:
			args[argument.Assignment.Name] = argument.Assignment.Value
		default:
			return nil, fmt.Errorf("Cannot marshal %v (arg %v)", f, argument)
		}
	}
	m[f.Name] = args
	return json.Marshal(m)
}

func (i *In) MarshalJSON() ([]byte, error) {
	switch {
	case i.FuncCall != nil:
		return json.Marshal(i.FuncCall)
	case i.Name != nil:
		return json.Marshal("$" + *i.Name)
	}
	return nil, fmt.Errorf("Cannot marshal %v", i)
}
