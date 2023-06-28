package model

import (
	"encoding/json"
	"errors"
	"time"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/logging"
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

	SolveFuncName           = "solve"
	SolveLegacyFuncName     = "solve_legacy"
	RequirementsKey         = "requirements"
	AtTimeKey               = "at_time"
	RequirementNameKey      = "name"
	RequirementNamespaceKey = "namespace"
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

type BuildExpression2 struct {
	Lets []Let
	Aps  []Ap
	Vars []Var
	AST  map[string]interface{}
}

type Ap struct {
	Name      string
	Arguments map[string]interface{}
}

type Let struct {
	Arguments map[string]interface{}
	InExpr    interface{}
}

type Var struct {
	Name  string
	Value interface{}
}

func NewBuildExpression2(data []byte) (*BuildExpression2, error) {
	expressionMap := make(map[string]interface{})
	err := json.Unmarshal(data, &expressionMap)
	if err != nil {
		return nil, errs.Wrap(err, "Could not unmarshal JSON")
	}

	// TODO: See if we can use a modified version of Mitchell's code to build the AST
	// If we can then discuss with the team how to integrate these changes together or
	// file a follow up story
	resultExpression := BuildExpression2{}
	err = traverse(expressionMap, &resultExpression)
	if err != nil {
		return nil, errs.Wrap(err, "Could not build expression")
	}

	return &resultExpression, nil
}

// func traverseForAst(expressionMap, ast map[string]interface{}) error {
// 	for key, value := range expressionMap {
// 		switch {
// 		case isLet(key, value):
// 			rawArguments, ok := value.(map[string]interface{})
// 			if !ok {
// 				return errs.New("Let value is not a map")
// 			}

// 			// Build up the AST from the let value
// 			arguments := make(map[string]interface{})
// 			err := traverseForAst(rawArguments, arguments)
// 			if err != nil {
// 				return errs.Wrap(err, "Could not build expression")
// 			}

// 			var inExpr interface{}
// 			rawInExpr, ok := rawArguments["in"].(map[string]interface{})
// 			if ok {
// 				inExpr = make(map[string]interface{})
// 				err = traverseForAst(rawInExpr, inExpr)
// 				if err != nil {
// 					return errs.Wrap(err, "Could not build expression")
// 				}
// 			}

// 			let := Let{
// 				Arguments: value.(map[string]interface{}),
// 				InExpr:    rawArguments["in"],
// 			}
// 		}
// 	}

// 	return nil
// }

func traverse(expressionMap map[string]interface{}, result *BuildExpression2) error {
	for key, value := range expressionMap {
		_, valueIsMap := value.(map[string]interface{})
		switch {
		case isLet(key, value):
			letVal, ok := value.(map[string]interface{})
			if !ok {
				return errs.New("Let value is not a map")
			}

			let := Let{
				Arguments: value.(map[string]interface{}),
				InExpr:    letVal["in"],
			}

			result.Lets = append(result.Lets, let)
			result.AST[key] = let

			err := traverse(value.(map[string]interface{}), result)
			if err != nil {
				return errs.Wrap(err, "Could not build expression")
			}
		case isAp(value):
			var apName string
			for k := range value.(map[string]interface{}) {
				apName = k
			}

			args, ok := value.(map[string]interface{})[apName].(map[string]interface{})
			if !ok {
				return errs.New("Ap arguments are not a map")
			}

			ap := Ap{
				Name:      apName,
				Arguments: args,
			}

			result.Aps = append(result.Aps, ap)
		case isVar(value):
			// May need some sort of context so we can identify the runtime variable with it's value
			// being the solve_legacy ap
			variable := Var{
				Name:  key,
				Value: value,
			}

			result.Vars = append(result.Vars, variable)
		case valueIsMap:
			err := traverse(value.(map[string]interface{}), result)
			if err != nil {
				return errs.Wrap(err, "Could not build expression")
			}
		}
	}

	return nil
}

func isLet(key string, value interface{}) bool {
	if key == "let" {
		letMap, ok := value.(map[string]interface{})
		if !ok {
			return false
		}
		if _, ok := letMap["in"]; !ok {
			return false
		}

		return true
	}
	return false
}

func isAp(value interface{}) bool {
	apMap, ok := value.(map[string]interface{})
	if !ok {
		return false
	}

	if len(apMap) != 1 {
		return false
	}

	var apName string
	for k := range apMap {
		apName = k
	}

	argMap, ok := apMap[apName].(map[string]interface{})
	if !ok {
		return false
	}

	if _, ok := argMap["in"]; ok {
		return false
	}

	return true
}

func isVar(value interface{}) bool {
	if _, ok := value.(string); ok {
		return true
	}

	return false
}

type Requirement struct {
	Name               string               `json:"name"`
	Namespace          string               `json:"namespace"`
	VersionRequirement []VersionRequirement `json:"version_requirements,omitempty"`
}

type VersionRequirement map[string]string

type BuildExpression struct {
	expression   map[string]interface{}
	solveNode    *map[string]interface{}
	requirements []Requirement
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
	expression := make(map[string]interface{})
	err := json.Unmarshal(data, &expression)
	if err != nil {
		return nil, errs.Wrap(err, "Could not unmarshal JSON")
	}

	solveNode, err := getSolveNode(expression)
	if err != nil {
		return nil, errs.Wrap(err, "Could not get solve node")
	}

	requirements, err := getRequirements(solveNode)
	if err != nil {
		return nil, errs.Wrap(err, "Could not get requirements node")
	}

	return &BuildExpression{
		expression:   expression,
		solveNode:    &solveNode,
		requirements: requirements,
	}, nil
}

// validateRequirements ensures that the requirements in the BuildExpression contain
// both the name and namespace fields. These fileds are used for requirement operations.
func validateRequirements(requirements []interface{}) error {
	for _, requirement := range requirements {
		r, ok := requirement.(map[string]interface{})
		if !ok {
			return errs.New("Requirement in BuildExpression is malformed")
		}

		_, ok = r[RequirementNameKey]
		if !ok {
			return errs.New("Requirement in BuildExpression is missing name field: %#v", r)
		}
		_, ok = r[RequirementNamespaceKey]
		if !ok {
			return errs.New("Requirement in BuildExpression is missing namespace field: %#v", r)
		}
	}
	return nil
}

// Requirements returns the requirements in the BuildExpression.
func (bx BuildExpression) Requirements() []Requirement {
	return bx.requirements
}

// Update updates the BuildExpression's requirements based on the operation and requirement.
func (bx *BuildExpression) Update(operation Operation, requirement Requirement, timestamp strfmt.DateTime) error {
	var err error
	switch operation {
	case OperationAdded:
		err = bx.addRequirement(requirement)
	case OperationRemoved:
		err = bx.removeRequirement(requirement)
	case OperationUpdated:
		err = bx.updateRequirement(requirement)
	default:
		return errs.New("Unsupported operation")
	}
	if err != nil {
		return errs.Wrap(err, "Could not update BuildExpression's requirements")
	}

	formatted, err := time.Parse(time.RFC3339, timestamp.String())
	if err != nil {
		return errs.Wrap(err, "Could not parse latest timestamp")
	}

	(*bx.solveNode)[AtTimeKey] = formatted

	return nil
}

// addRequirement adds a requirement to the BuildExpression.
func (bx *BuildExpression) addRequirement(requirement Requirement) error {
	bx.requirements = append(bx.requirements, requirement)

	(*bx.solveNode)[RequirementsKey] = bx.requirements

	return nil
}

// removeRequirement removes a requirement from the BuildExpression.
func (bx *BuildExpression) removeRequirement(requirement Requirement) error {
	for i, req := range bx.requirements {
		if req.Name == requirement.Name && req.Namespace == requirement.Namespace {
			bx.requirements = append(bx.requirements[:i], bx.requirements[i+1:]...)
			(*bx.solveNode)[RequirementsKey] = bx.requirements
			return nil
		}
	}

	return errs.New("Could not find requirement")
}

// updateRequirement updates an existing requirement in the BuildExpression.
func (bx BuildExpression) updateRequirement(requirement Requirement) error {
	for i, req := range bx.requirements {
		if req.Name == requirement.Name && req.Namespace == requirement.Namespace {
			bx.requirements[i] = requirement
			(*bx.solveNode)[RequirementsKey] = bx.requirements
			return nil
		}
	}

	return errs.New("Could not find requirement")
}

func (bx BuildExpression) MarshalJSON() ([]byte, error) {
	return json.Marshal(bx.expression)
}

// getRequirements returns the list of requirements from the solve node of the build expression.
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
func getRequirements(solveNode map[string]interface{}) ([]Requirement, error) {
	for k, v := range solveNode {
		if k != RequirementsKey {
			continue
		}

		node, ok := v.([]interface{})
		if !ok {
			return nil, errs.New("Requirements in BuildExpression are malformed")
		}

		requirementsData, err := json.Marshal(node)
		if err != nil {
			return nil, errs.Wrap(err, "Could not marshal JSON")
		}

		var requirements []Requirement
		err = json.Unmarshal(requirementsData, &requirements)
		if err != nil {
			return nil, errs.Wrap(err, "Could not unmarshal JSON")
		}

		err = validateRequirements(node)
		if err != nil {
			return nil, errs.Wrap(err, "Requirements in BuildExpression are invalid")
		}

		return requirements, nil
	}

	return nil, errs.New("Could not find requirements node")
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
func getSolveNode(expression map[string]interface{}) (map[string]interface{}, error) {
	solveNode, err := getFuncNode(expression, SolveFuncName)
	if err == nil {
		return solveNode, nil
	}
	if !errors.Is(err, funcNodeNotFoundError) {
		return nil, errs.Wrap(err, "Could not get solve node")
	}
	logging.Debug("Could not get solve node, trying solve legacy node")

	return getFuncNode(expression, SolveLegacyFuncName)
}

// getFuncNode returns the node of the given function name from the build expression.
// It returns an error if the function node is not found.
// Currently, this function just recurses the build expression until it finds the function node
// of the correct map[string]interface{} type.
// It expects the JSON representation of the build expression to be formatted as follows:
//
//	{
//	  "let": {
//	    "runtime": {
//	      "func_name": {
//	      }
//	    }
//	  }
//	}
func getFuncNode(expression map[string]interface{}, funcName string) (map[string]interface{}, error) {
	for k, v := range expression {
		node, ok := v.(map[string]interface{})
		if !ok {
			continue
		}

		if k == funcName {
			return node, nil
		}

		// We recurse the build expression until we find the function node
		if childNode, err := getFuncNode(node, funcName); err == nil {
			return childNode, nil
		}
	}

	return nil, funcNodeNotFoundError
}
