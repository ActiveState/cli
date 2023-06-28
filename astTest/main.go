package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"github.com/ActiveState/cli/internal/environment"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/rtutils/p"
)

const (
	ctxLet         = "let"
	ctxIn          = "in"
	ctxAp          = "ap"
	ctxValue       = "value"
	ctxAssignments = "assignments"
	ctxIsAp        = "isAp"
)

type Expression struct {
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

type Requirement struct {
	Name               string               `json:"name"`
	Namespace          string               `json:"namespace"`
	VersionRequirement []VersionRequirement `json:"version_requirements,omitempty"`
}

type VersionRequirement map[string]string

func main() {
	if err := run(); err != nil {
		panic(err)
	}
}

func run() error {
	wd, err := environment.GetRootPath()
	if err != nil {
		return err
	}

	data, err := fileutils.ReadFile(filepath.Join(wd, "pkg", "platform", "api", "buildplanner", "model", "testdata", "buildexpression.json"))
	if err != nil {
		return err
	}

	expr, err := NewBuildExpression(data)
	if err != nil {
		return err
	}

	exprData, err := expr.MarshalJSON()
	if err != nil {
		return errs.Wrap(err, "Could not marshal expression")
	}

	// exprData, err := json.MarshalIndent(expr, "", "  ")
	// if err != nil {
	// 	return errs.Wrap(err, "Could not marshal expression")
	// }
	fmt.Println(string(exprData))

	reqs := expr.Requirements()
	reqsData, err := json.MarshalIndent(reqs, "", "  ")
	if err != nil {
		return errs.Wrap(err, "Could not marshal requirements")
	}
	fmt.Println(string(reqsData))
	return nil
}

func NewBuildExpression(data []byte) (*Expression, error) {
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

	return &Expression{let}, nil
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
			b, err := json.Marshal(v)
			if err != nil {
				return nil, errs.Wrap(err, "Could not marshal string '%s'", v)
			}
			value.Str = p.StrP(string(b))
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

func (e *Expression) Requirements() []Requirement {
	var solveAp *Ap
	for _, a := range e.Let.Assignments {
		if a.Value.Ap == nil {
			continue
		}

		if a.Value.Ap.Name == "solve" || a.Value.Ap.Name == "solve_legacy" {
			solveAp = a.Value.Ap
		}
	}

	var reqs []*Value
	for _, arg := range solveAp.Arguments {
		if arg.Assignment == nil {
			continue
		}

		if arg.Assignment.Name == "requirements" && arg.Assignment.Value != nil {
			reqs = *arg.Assignment.Value.List
		}
	}

	var requirements []Requirement
	for _, r := range reqs {
		if r.Object == nil {
			continue
		}

		var req Requirement
		for _, o := range *r.Object {
			if o.Name == "name" {
				req.Name = *o.Value.Str
			}

			if o.Name == "namespace" {
				req.Namespace = *o.Value.Str
			}

			if o.Name == "version_requirements" {
				req.VersionRequirement = getVersionRequirements(o.Value.List)
			}
		}
		requirements = append(requirements, req)
	}

	return requirements
}

func getVersionRequirements(v *[]*Value) []VersionRequirement {
	var reqs []VersionRequirement
	for _, r := range *v {
		if r.Object == nil {
			continue
		}

		versionReq := make(VersionRequirement)
		for _, o := range *r.Object {
			if o.Name == "comparator" {
				versionReq["comparator"] = *o.Value.Str
			}

			if o.Name == "version" {
				versionReq["version"] = *o.Value.Str
			}
		}
		reqs = append(reqs, versionReq)
	}
	return reqs
}

func (e *Expression) MarshalJSON() ([]byte, error) {
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
			return nil, errors.New(fmt.Sprintf("Cannot marshal %v (arg %v)", f, argument))
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
	return nil, errors.New(fmt.Sprintf("Cannot marshal %v", i))
}
