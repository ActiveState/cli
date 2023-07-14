package buildexpression

import (
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/multilog"
	"github.com/ActiveState/cli/internal/rtutils/ptr"
	"github.com/ActiveState/cli/internal/sliceutils"
	"github.com/ActiveState/cli/pkg/platform/api/buildplanner/model"
	"github.com/go-openapi/strfmt"
)

const (
	SolveFuncName                     = "solve"
	SolveLegacyFuncName               = "solve_legacy"
	RequirementsKey                   = "requirements"
	PlatformsKey                      = "platforms"
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

var funcNodeNotFoundError = errors.New("Could not find function node")

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
func New(data []byte) (*BuildExpression, error) {
	rawBuildExpression := make(map[string]interface{})
	err := json.Unmarshal(data, &rawBuildExpression)
	if err != nil {
		return nil, errs.Wrap(err, "Could not unmarshal build expression")
	}

	letValue, ok := rawBuildExpression["let"]
	if !ok {
		return nil, errs.New("Build expression has no 'let' key")
	}

	letMap, ok := letValue.(map[string]interface{})
	if !ok {
		return nil, errs.New("'let' key is not a JSON object")
	}

	var path []string
	let, err := newLet(path, letMap)
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

func newLet(path []string, m map[string]interface{}) (*Let, error) {
	path = append(path, ctxLet)
	defer func() {
		_, _, err := sliceutils.Pop(path)
		if err != nil {
			multilog.Error("Could not pop context: %v", err)
		}
	}()

	inValue, ok := m["in"]
	if !ok {
		return nil, errs.New("Build expression's 'let' object has no 'in' key")
	}

	in, err := newIn(path, inValue)
	if err != nil {
		return nil, errs.Wrap(err, "Could not parse 'in' key's value: %v", inValue)
	}

	// Delete in so it doesn't get parsed as an assignment.
	delete(m, "in")

	assignments, err := newAssignments(path, m)
	if err != nil {
		return nil, errs.Wrap(err, "Could not parse 'let' key")
	}

	return &Let{Assignments: *assignments, In: in}, nil
}

func isAp(path []string, value map[string]interface{}) bool {
	path = append(path, ctxIsAp)
	defer func() {
		_, _, err := sliceutils.Pop(path)
		if err != nil {
			multilog.Error("Could not pop context: %v", err)
		}
	}()

	_, hasIn := value["in"]
	if hasIn && !sliceutils.Contains(path, ctxAssignments) {
		return false
	}

	return true
}

func newValue(path []string, valueInterface interface{}) (*Value, error) {
	path = append(path, ctxValue)
	defer func() {
		_, _, err := sliceutils.Pop(path)
		if err != nil {
			multilog.Error("Could not pop context: %v", err)
		}
	}()

	value := &Value{}

	switch v := valueInterface.(type) {
	case map[string]interface{}:
		// Examine keys first to see if this is a function call.
		for key, val := range v {
			if _, ok := val.(map[string]interface{}); !ok {
				continue
			}

			if isAp(path, val.(map[string]interface{})) {
				f, err := newAp(path, v)
				if err != nil {
					return nil, errs.Wrap(err, "Could not parse '%s' function's value: %v", key, v)
				}
				value.Ap = f
			}
		}

		if value.Ap == nil {
			// It's not a function call, but an object.
			object, err := newAssignments(path, v)
			if err != nil {
				return nil, errs.Wrap(err, "Could not parse object: %v", v)
			}
			value.Object = object
		}

	case []interface{}:
		values := []*Value{}
		for _, item := range v {
			value, err := newValue(path, item)
			if err != nil {
				return nil, errs.Wrap(err, "Could not parse list: %v", v)
			}
			values = append(values, value)
		}
		value.List = &values

	case string:
		if sliceutils.Contains(path, ctxIn) {
			value.Ident = &v
		} else {
			value.Str = ptr.To(v)
		}

	default:
		// An empty value is interpreted as JSON null.
		value.Null = &Null{}
	}

	return value, nil
}

func newAp(path []string, m map[string]interface{}) (*Ap, error) {
	path = append(path, ctxAp)
	defer func() {
		_, _, err := sliceutils.Pop(path)
		if err != nil {
			multilog.Error("Could not pop context: %v", err)
		}
	}()

	// Look in the given object for the function's name and argument object or list.
	var name string
	var argsInterface interface{}
	for key, value := range m {
		if isAp(path, value.(map[string]interface{})) {
			name = key
			argsInterface = value
			break
		}
	}

	args := []*Value{}

	switch v := argsInterface.(type) {
	case map[string]interface{}:
		for key, valueInterface := range v {
			value, err := newValue(path, valueInterface)
			if err != nil {
				return nil, errs.Wrap(err, "Could not parse '%s' function's argument '%s': %v", name, key, valueInterface)
			}
			args = append(args, &Value{Assignment: &Var{Name: key, Value: value}})
		}
		sort.SliceStable(args, func(i, j int) bool { return args[i].Assignment.Name < args[j].Assignment.Name })

	case []interface{}:
		for _, item := range v {
			value, err := newValue(path, item)
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

func newAssignments(path []string, m map[string]interface{}) (*[]*Var, error) {
	path = append(path, ctxAssignments)
	defer func() {
		_, _, err := sliceutils.Pop(path)
		if err != nil {
			multilog.Error("Could not pop context: %v", err)
		}
	}()

	assignments := []*Var{}
	for key, valueInterface := range m {
		value, err := newValue(path, valueInterface)
		if err != nil {
			return nil, errs.Wrap(err, "Could not parse '%s' key's value: %v", key, valueInterface)
		}
		assignments = append(assignments, &Var{Name: key, Value: value})
	}
	sort.SliceStable(assignments, func(i, j int) bool { return assignments[i].Name < assignments[j].Name })
	return &assignments, nil
}

func newIn(path []string, inValue interface{}) (*In, error) {
	path = append(path, ctxIn)
	defer func() {
		_, _, err := sliceutils.Pop(path)
		if err != nil {
			multilog.Error("Could not pop context: %v", err)
		}
	}()

	in := &In{}

	switch v := inValue.(type) {
	case map[string]interface{}:
		f, err := newAp(path, v)
		if err != nil {
			return nil, errs.Wrap(err, "'in' object is not a function call")
		}
		in.FuncCall = f

	case string:
		in.Name = ptr.To(strings.TrimPrefix(v, "$"))

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

			if o.Name == RequirementNameKey || o.Name == RequirementNamespaceKey {
				if o.Value.Str == nil {
					return errs.New("Requirement object value is not set to a string")
				}
			}

			if o.Name == RequirementVersionRequirementsKey {
				if o.Value.List == nil {
					return errs.New("Requirement object value is not set to a list")
				}
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
func (e *BuildExpression) Requirements() []model.Requirement {
	requirementsNode := e.getRequirementsNode()

	var requirements []model.Requirement
	for _, r := range requirementsNode {
		if r.Object == nil {
			continue
		}

		var req model.Requirement
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

func getVersionRequirements(v *[]*Value) []model.VersionRequirement {
	var reqs []model.VersionRequirement

	if v == nil {
		return reqs
	}

	for _, r := range *v {
		if r.Object == nil {
			continue
		}

		versionReq := make(model.VersionRequirement)
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

func (e *BuildExpression) getSolveNodeArguments() []*Value {
	solveAp := e.getSolveNode()
	if solveAp == nil {
		return []*Value{}
	}

	return solveAp.Arguments
}

func (e *BuildExpression) getPlatformsNode() *[]*Value {
	solveAp := e.getSolveNode()
	if solveAp == nil {
		return nil
	}

	for _, arg := range solveAp.Arguments {
		if arg.Assignment == nil {
			continue
		}

		if arg.Assignment.Name == PlatformsKey && arg.Assignment.Value != nil {
			return arg.Assignment.Value.List
		}
	}

	return nil
}

// Update updates the BuildExpression's requirements based on the operation and requirement.
func (e *BuildExpression) UpdateRequirement(operation model.Operation, requirement model.Requirement) error {
	var err error
	switch operation {
	case model.OperationAdded:
		err = e.addRequirement(requirement)
	case model.OperationRemoved:
		err = e.removeRequirement(requirement)
	case model.OperationUpdated:
		err = e.updateRequirement(requirement)
	default:
		return errs.New("Unsupported operation")
	}
	if err != nil {
		return errs.Wrap(err, "Could not update BuildExpression's requirements")
	}

	return nil
}

func (e *BuildExpression) addRequirement(requirement model.Requirement) error {
	obj := []*Var{
		{Name: RequirementNameKey, Value: &Value{Str: ptr.To(requirement.Name)}},
		{Name: RequirementNamespaceKey, Value: &Value{Str: ptr.To(requirement.Namespace)}},
	}

	if requirement.VersionRequirement != nil {
		for _, r := range requirement.VersionRequirement {
			obj = append(obj, &Var{Name: RequirementVersionRequirementsKey, Value: &Value{List: &[]*Value{
				{Object: &[]*Var{
					{Name: RequirementComparatorKey, Value: &Value{Str: ptr.To(r[RequirementComparatorKey])}},
					{Name: RequirementVersionKey, Value: &Value{Str: ptr.To(r[RequirementVersionKey])}},
				}}},
			}})
		}
	}

	requirementsNode := append(e.getRequirementsNode(), &Value{Object: &obj})

	for _, arg := range e.getSolveNodeArguments() {
		if arg.Assignment == nil {
			continue
		}

		if arg.Assignment.Name == RequirementsKey {
			arg.Assignment.Value.List = &requirementsNode
		}
	}

	return nil
}

func (e *BuildExpression) removeRequirement(requirement model.Requirement) error {
	requirementsNode := e.getRequirementsNode()

	var found bool
	for i, r := range requirementsNode {
		if r.Object == nil {
			continue
		}

		for _, o := range *r.Object {
			if o.Name == RequirementNameKey && *o.Value.Str == requirement.Name {
				requirementsNode = append(requirementsNode[:i], requirementsNode[i+1:]...)
				found = true
				break
			}
		}
	}

	if !found {
		return errs.New("Could not find requirement")
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

func (e *BuildExpression) updateRequirement(requirement model.Requirement) error {
	if requirement.VersionRequirement == nil {
		return nil
	}

	requirementsNode := e.getRequirementsNode()

	for _, r := range requirementsNode {
		if r.Object == nil {
			continue
		}

		for _, o := range *r.Object {
			if o.Name != RequirementNameKey || *o.Value.Str != requirement.Name {
				continue
			}

			var versionRequirements []*Value
			for _, v := range *r.Object {
				if v.Name != RequirementVersionRequirementsKey {
					continue
				}

				for _, versionReq := range requirement.VersionRequirement {
					versionRequirements = append(versionRequirements, &Value{Object: &[]*Var{
						{Name: RequirementComparatorKey, Value: &Value{Str: ptr.To(versionReq[RequirementComparatorKey])}},
						{Name: RequirementVersionKey, Value: &Value{Str: ptr.To(versionReq[RequirementVersionKey])}},
					}})
				}
				v.Value.List = &versionRequirements
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

func (e *BuildExpression) UpdatePlatform(operation model.Operation, platformID strfmt.UUID) error {
	var err error
	switch operation {
	case model.OperationAdded:
		err = e.addPlatform(platformID)
	case model.OperationRemoved:
		err = e.removePlatform(platformID)
	default:
		return errs.New("Unsupported operation")
	}
	if err != nil {
		return errs.Wrap(err, "Could not update BuildExpression's platform")
	}

	return nil
}

func (e *BuildExpression) addPlatform(platformID strfmt.UUID) error {
	platformsNode := e.getPlatformsNode()

	*platformsNode = append(*platformsNode, &Value{Str: ptr.To(platformID.String())})

	return nil
}

func (e *BuildExpression) removePlatform(platformID strfmt.UUID) error {
	platformsNode := e.getPlatformsNode()

	var found bool
	for i, p := range *platformsNode {
		if p.Str == nil {
			continue
		}

		if *p.Str == platformID.String() {
			*platformsNode = append((*platformsNode)[:i], (*platformsNode)[i+1:]...)
			found = true
			break
		}
	}

	if !found {
		return errs.New("Could not find platform")
	}

	return nil
}

func (e *BuildExpression) UpdateTimestamp(timestamp strfmt.DateTime) error {
	formatted, err := time.Parse(time.RFC3339, timestamp.String())
	if err != nil {
		return errs.Wrap(err, "Could not parse latest timestamp")
	}

	for _, arg := range e.getSolveNode().Arguments {
		if arg.Assignment == nil {
			continue
		}

		if arg.Assignment.Name == "at_time" {
			arg.Assignment.Value.Str = ptr.To(formatted.Format(time.RFC3339))
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

	let["in"] = e.Let.In
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
	return json.Marshal([]*Value{})
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
