package buildexpression

import (
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/multilog"
	"github.com/ActiveState/cli/internal/rtutils/ptr"
	"github.com/ActiveState/cli/internal/sliceutils"
	"github.com/ActiveState/cli/pkg/platform/api/buildplanner/types"
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
	RequirementRevisionKey            = "revision"
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
	Let         *Let
	Assignments []*Value
}

type Let struct {
	// Let statements can be nested.
	// Each let will contain its own assignments and an in statement.
	Let         *Let
	Assignments []*Var
	In          *In
}

type Var struct {
	Name  string
	Value *Value
}

type Value struct {
	Ap    *Ap
	List  *[]*Value
	Str   *string
	Null  *Null
	Float *float64
	Int   *int

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

// Unmarshal creates a BuildExpression from a JSON byte array.
func Unmarshal(data []byte) (*BuildExpression, error) {
	rawBuildExpression := make(map[string]interface{})
	err := json.Unmarshal(data, &rawBuildExpression)
	if err != nil {
		return nil, errs.Wrap(err, "Could not unmarshal build expression")
	}

	if len(rawBuildExpression) != 1 {
		return nil, errs.New("Build expression must have exactly one key")
	}

	expr := &BuildExpression{}
	var path []string
	for key, value := range rawBuildExpression {
		switch v := value.(type) {
		case map[string]interface{}:
			// At this level the key must either be a let, an ap, or an assignment.
			if key == "let" {
				let, err := newLet(path, v)
				if err != nil {
					return nil, errs.Wrap(err, "Could not parse 'let' key")
				}

				expr.Let = let
			} else if isAp(path, v) {
				ap, err := newAp(path, v)
				if err != nil {
					return nil, errs.Wrap(err, "Could not parse '%s' key", key)
				}

				expr.Assignments = append(expr.Assignments, &Value{Ap: ap})
			} else {
				assignments, err := newAssignments(path, v)
				if err != nil {
					return nil, errs.Wrap(err, "Could not parse assignments")
				}

				expr.Assignments = append(expr.Assignments, &Value{Assignment: &Var{Name: key, Value: &Value{Object: &assignments}}})
			}
		default:
			return nil, errs.New("Build expression's value must be a map[string]interface{}")
		}
	}

	err = expr.validateRequirements()
	if err != nil {
		return nil, errs.Wrap(err, "Could not validate requirements")
	}

	err = expr.normalizeTimestamp()
	if err != nil {
		return nil, errs.Wrap(err, "Could not normalize timestamp")
	}

	return expr, nil
}

// New creates a minimal, empty buildexpression.
func New() (*BuildExpression, error) {
	// At this time, there is no way to ask the Platform for an empty buildexpression, so build one
	// manually.
	expr, err := Unmarshal([]byte(`
{
	"let": {
		"sources": {
				"solve": {
					"at_time": "$at_time",
					"platforms": [],
					"requirements": [],
					"solver_version": null
				}
		},
		"runtime": {
				"state_tool_artifacts": {
						"src": "$sources"
				}
		},
		"in": "$runtime"
	}
}`))
	if err != nil {
		return nil, errs.Wrap(err, "Unable to create initial buildexpression")
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

	result := &Let{In: in}
	let, ok := m["let"]
	if ok {
		letMap, ok := let.(map[string]interface{})
		if !ok {
			return nil, errs.New("'let' key's value is not a map[string]interface{}")
		}

		l, err := newLet(path, letMap)
		if err != nil {
			return nil, errs.Wrap(err, "Could not parse 'let' key")
		}
		result.Let = l

		// Delete let so it doesn't get parsed as an assignment.
		delete(m, "let")
	}

	assignments, err := newAssignments(path, m)
	if err != nil {
		return nil, errs.Wrap(err, "Could not parse assignments")
	}

	result.Assignments = assignments
	return result, nil
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

			// If the length of the value is greater than 1,
			// then it's not a function call. It's an object
			// and will be set as such outside the loop.
			if len(v) > 1 {
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
			value.Object = &object
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

	case float64:
		value.Float = ptr.To(v)

	case nil:
		// An empty value is interpreted as JSON null.
		value.Null = &Null{}

	default:
		logging.Debug("Unknown type: %T at path %s", v, strings.Join(path, "."))
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

	// m is a mapping of function name to arguments. There should only be one
	// set of arugments. Since the arguments are key-value pairs, it should be
	// a map[string]interface{}.
	if len(m) > 1 {
		return nil, errs.New("Function call has more than one argument mapping")
	}

	// Look in the given object for the function's name and argument mapping.
	var name string
	var argsInterface interface{}
	for key, value := range m {
		_, ok := value.(map[string]interface{})
		if !ok {
			return nil, errs.New("Incorrect argument format")
		}

		name = key
		argsInterface = value
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

func newAssignments(path []string, m map[string]interface{}) ([]*Var, error) {
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
	sort.SliceStable(assignments, func(i, j int) bool {
		return assignments[i].Name < assignments[j].Name
	})
	return assignments, nil
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
	requirements, err := e.getRequirementsNode()
	if err != nil {
		return errs.Wrap(err, "Could not get requirements node")
	}

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

func (e *BuildExpression) Requirements() ([]types.Requirement, error) {
	requirementsNode, err := e.getRequirementsNode()
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

	return requirements, nil
}

func (e *BuildExpression) getRequirementsNode() ([]*Value, error) {
	solveAp, err := e.getSolveNode()
	if err != nil {
		return nil, errs.Wrap(err, "Could not get solve node")
	}

	var reqs []*Value
	for _, arg := range solveAp.Arguments {
		if arg.Assignment == nil {
			continue
		}

		if arg.Assignment.Name == RequirementsKey && arg.Assignment.Value != nil {
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
func (e *BuildExpression) getSolveNode() (*Ap, error) {
	// First, try to find the solve node via lets.
	if e.Let != nil {
		solveAp, err := recurseLets(e.Let)
		if err != nil {
			return nil, errs.Wrap(err, "Could not recurse lets")
		}

		return solveAp, nil
	}

	// Search for solve node in the top level assignments.
	for _, a := range e.Assignments {
		if a.Assignment == nil {
			continue
		}

		if a.Assignment.Name == "" {
			continue
		}

		if a.Assignment.Value == nil {
			continue
		}

		if a.Assignment.Value.Ap == nil {
			continue
		}

		if a.Assignment.Value.Ap.Name == SolveFuncName || a.Assignment.Value.Ap.Name == SolveLegacyFuncName {
			return a.Assignment.Value.Ap, nil
		}
	}

	return nil, funcNodeNotFoundError
}

// recurseLets recursively searches for the solve node in the let statements.
// The solve node is specified by the name "runtime" and the function name "solve"
// or "solve_legacy".
func recurseLets(let *Let) (*Ap, error) {
	for _, a := range let.Assignments {
		if a.Value == nil {
			continue
		}

		if a.Value.Ap == nil {
			continue
		}

		if a.Name == "" {
			continue
		}

		if a.Value.Ap.Name == SolveFuncName || a.Value.Ap.Name == SolveLegacyFuncName {
			return a.Value.Ap, nil
		}
	}

	// The highest level solve node is not found, so recurse into the next let.
	if let.Let != nil {
		return recurseLets(let.Let)
	}

	return nil, funcNodeNotFoundError
}

func (e *BuildExpression) getSolveNodeArguments() ([]*Value, error) {
	solveAp, err := e.getSolveNode()
	if err != nil {
		return nil, errs.Wrap(err, "Could not get solve node")
	}

	return solveAp.Arguments, nil
}

func (e *BuildExpression) getSolveAtTimeValue() (*Value, error) {
	solveAp, err := e.getSolveNode()
	if err != nil {
		return nil, errs.Wrap(err, "Could not get solve node")
	}

	for _, arg := range solveAp.Arguments {
		if arg.Assignment != nil && arg.Assignment.Name == AtTimeKey {
			return arg.Assignment.Value, nil
		}
	}

	return nil, errs.New("Could not find %s", AtTimeKey)
}

func (e *BuildExpression) getPlatformsNode() (*[]*Value, error) {
	solveAp, err := e.getSolveNode()
	if err != nil {
		return nil, errs.Wrap(err, "Could not get solve node")
	}

	for _, arg := range solveAp.Arguments {
		if arg.Assignment == nil {
			continue
		}

		if arg.Assignment.Name == PlatformsKey && arg.Assignment.Value != nil {
			return arg.Assignment.Value.List, nil
		}
	}

	return nil, errs.New("Could not find platforms node")
}

func (e *BuildExpression) UpdateRequirement(operation types.Operation, requirement types.Requirement) error {
	var err error
	switch operation {
	case types.OperationAdded:
		err = e.addRequirement(requirement)
	case types.OperationRemoved:
		err = e.removeRequirement(requirement)
	case types.OperationUpdated:
		err = e.removeRequirement(requirement)
		if err != nil {
			break
		}
		err = e.addRequirement(requirement)
	default:
		return errs.New("Unsupported operation")
	}
	if err != nil {
		return errs.Wrap(err, "Could not update BuildExpression's requirements")
	}

	return nil
}

func (e *BuildExpression) addRequirement(requirement types.Requirement) error {
	obj := []*Var{
		{Name: RequirementNameKey, Value: &Value{Str: ptr.To(requirement.Name)}},
		{Name: RequirementNamespaceKey, Value: &Value{Str: ptr.To(requirement.Namespace)}},
	}

	if requirement.Revision != nil {
		obj = append(obj, &Var{Name: RequirementRevisionKey, Value: &Value{Int: requirement.Revision}})
	}

	if requirement.VersionRequirement != nil {
		values := []*Value{}
		for _, r := range requirement.VersionRequirement {
			values = append(values, &Value{Object: &[]*Var{
				{Name: RequirementComparatorKey, Value: &Value{Str: ptr.To(r[RequirementComparatorKey])}},
				{Name: RequirementVersionKey, Value: &Value{Str: ptr.To(r[RequirementVersionKey])}},
			}})
		}
		obj = append(obj, &Var{Name: RequirementVersionRequirementsKey, Value: &Value{List: &values}})
	}

	requirementsNode, err := e.getRequirementsNode()
	if err != nil {
		return errs.Wrap(err, "Could not get requirements node")
	}

	requirementsNode = append(requirementsNode, &Value{Object: &obj})

	arguments, err := e.getSolveNodeArguments()
	if err != nil {
		return errs.Wrap(err, "Could not get solve node arguments")
	}

	for _, arg := range arguments {
		if arg.Assignment == nil {
			continue
		}

		if arg.Assignment.Name == RequirementsKey {
			arg.Assignment.Value.List = &requirementsNode
		}
	}

	return nil
}

type RequirementNotFoundError struct {
	Name                   string
	*locale.LocalizedError // for legacy non-user-facing error usages
}

func (e *BuildExpression) removeRequirement(requirement types.Requirement) error {
	requirementsNode, err := e.getRequirementsNode()
	if err != nil {
		return errs.Wrap(err, "Could not get requirements node")
	}

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
		return &RequirementNotFoundError{
			requirement.Name,
			locale.NewInputError("err_remove_requirement_not_found", "", requirement.Name),
		}
	}

	solveNode, err := e.getSolveNode()
	if err != nil {
		return errs.Wrap(err, "Could not get solve node")
	}

	for _, arg := range solveNode.Arguments {
		if arg.Assignment == nil {
			continue
		}

		if arg.Assignment.Name == RequirementsKey {
			arg.Assignment.Value.List = &requirementsNode
		}
	}

	return nil
}

func (e *BuildExpression) UpdatePlatform(operation types.Operation, platformID strfmt.UUID) error {
	var err error
	switch operation {
	case types.OperationAdded:
		err = e.addPlatform(platformID)
	case types.OperationRemoved:
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
	platformsNode, err := e.getPlatformsNode()
	if err != nil {
		return errs.Wrap(err, "Could not get platforms node")
	}

	*platformsNode = append(*platformsNode, &Value{Str: ptr.To(platformID.String())})

	return nil
}

func (e *BuildExpression) removePlatform(platformID strfmt.UUID) error {
	platformsNode, err := e.getPlatformsNode()
	if err != nil {
		return errs.Wrap(err, "Could not get platforms node")
	}

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

func (e *BuildExpression) SetDefaultAtTime() error {
	atTimeNode, err := e.getSolveAtTimeValue()
	if err != nil {
		return errs.Wrap(err, "Could not get %s node", AtTimeKey)
	}
	atTimeNode.Str = ptr.To("$" + AtTimeKey)
	return nil
}

func (e *BuildExpression) MaybeSetDefaultAtTime(ts *time.Time) error {
	if ts == nil {
		return nil // nothing to compare to
	}
	atTimeNode, err := e.getSolveAtTimeValue()
	if err != nil {
		return errs.Wrap(err, "Could not get %s node", AtTimeKey)
	}
	if strings.HasPrefix(*atTimeNode.Str, "$") {
		return nil
	}
	if atTime, err := strfmt.ParseDateTime(*atTimeNode.Str); err == nil && time.Time(atTime).Equal(*ts) {
		return e.SetDefaultAtTime()
	} else if err != nil {
		return errs.Wrap(err, "Invalid timestamp: %s", *atTimeNode.Str)
	}
	return nil
}

// normalizeTimestamp normalizes the solve node's timestamp, if possible.
// Platform timestamps may differ from the strfmt.DateTime format. For example, Platform
// timestamps will have microsecond precision, while strfmt.DateTime will only have millisecond
// precision. This will affect comparisons between buildexpressions (which is normally done
// byte-by-byte).
func (e *BuildExpression) normalizeTimestamp() error {
	atTimeNode, err := e.getSolveAtTimeValue()
	if err != nil {
		return errs.Wrap(err, "Could not get at time node")
	}

	if atTimeNode.Str != nil && !strings.HasPrefix(*atTimeNode.Str, "$") {
		atTime, err := strfmt.ParseDateTime(*atTimeNode.Str)
		if err != nil {
			return errs.Wrap(err, "Invalid timestamp: %s", *atTimeNode.Str)
		}
		atTimeNode.Str = ptr.To(atTime.String())
	}

	return nil
}

func (e *BuildExpression) Copy() (*BuildExpression, error) {
	bytes, err := json.Marshal(e)
	if err != nil {
		return nil, errs.Wrap(err, "Failed to marshal build expression during copy")
	}
	return Unmarshal(bytes)
}

func (e *BuildExpression) MarshalJSON() ([]byte, error) {
	m := make(map[string]interface{})

	if e.Let != nil {
		m["let"] = e.Let
	}

	for _, value := range e.Assignments {
		if value.Assignment == nil {
			continue
		}

		m[value.Assignment.Name] = value
	}

	return json.Marshal(m)
}

func (l *Let) MarshalJSON() ([]byte, error) {
	m := make(map[string]interface{})

	if l.Let != nil {
		m["let"] = l.Let
	}

	for _, v := range l.Assignments {
		if v.Value == nil {
			continue
		}

		m[v.Name] = v.Value
	}

	m["in"] = l.In

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
	case v.Float != nil:
		return json.Marshal(*v.Float)
	case v.Int != nil:
		return json.Marshal(*v.Int)
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
