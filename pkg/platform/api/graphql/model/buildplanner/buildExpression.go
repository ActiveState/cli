package model

import (
	"encoding/json"
	"errors"
	"time"

	"github.com/ActiveState/cli/internal/condition"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/pkg/platform/api/inventory"
	"github.com/ActiveState/cli/pkg/platform/api/inventory/inventory_client/inventory_operations"
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
	raw          json.RawMessage
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
		raw:          data,
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
func (bx *BuildExpression) Update(operation Operation, requirement Requirement) error {
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

	err = bx.UpdateTimestamp()
	if err != nil {
		return errs.Wrap(err, "Could not update BuildExpression's timestamp")
	}

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

// UpdateTimestamp fetches the latest platform timestamp and updates it in the BuildExpression.
func (bx BuildExpression) UpdateTimestamp() error {
	latest, err := fetchLatestTimeStamp()
	if err != nil {
		return errs.Wrap(err, "Could not fetch latest timestamp")
	}

	formatted, err := time.Parse(time.RFC3339, latest.String())
	if err != nil {
		return errs.Wrap(err, "Could not parse latest timestamp")
	}

	(*bx.solveNode)[AtTimeKey] = formatted
	return nil
}

func (bx BuildExpression) MarshalJSON() ([]byte, error) {
	return json.Marshal(bx.expression)
}

// fetchLatestTimeStamp fetches the latest timestamp from the inventory service.
// This function lives in this package to avoid an import cycle.
func fetchLatestTimeStamp() (*strfmt.DateTime, error) {
	if condition.InTest() {
		return &strfmt.DateTime{}, nil
	}

	client := inventory.Get()
	result, err := client.GetLatestTimestamp(inventory_operations.NewGetLatestTimestampParams())
	if err != nil {
		return nil, errs.Wrap(err, "GetLatestTimestamp failed")
	}

	return result.Payload.Timestamp, nil
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

func (bx *BuildExpression) String() string {
	return string(bx.raw)
}
