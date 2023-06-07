package model

import (
	"encoding/json"
	"errors"
	"time"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/logging"
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
}

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

	requirementsNode, err := getRequirementsNode(expression)
	if err != nil {
		return nil, errs.Wrap(err, "Could not get requirements node")
	}

	requirementsData, err := json.Marshal(requirementsNode)
	if err != nil {
		return nil, errs.Wrap(err, "Could not marshal JSON")
	}

	var requirements []Requirement
	err = json.Unmarshal(requirementsData, &requirements)
	if err != nil {
		return nil, errs.Wrap(err, "Could not unmarshal JSON")
	}

	err = validateRequirements(requirementsNode)
	if err != nil {
		return nil, errs.Wrap(err, "Requirements in BuildExpression are invalid")
	}

	return &BuildExpression{
		expression:   expression,
		solveNode:    &solveNode,
		requirements: requirements,
	}, nil
}

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

func (bx BuildExpression) Requirements() []Requirement {
	return bx.requirements
}

func (bx *BuildExpression) Update(operation Operation, requirement Requirement) error {
	var err error
	switch operation {
	case OperationAdded:
		err = bx.AddRequirement(requirement)
	case OperationRemoved:
		err = bx.RemoveRequirement(requirement)
	case OperationUpdated:
		err = bx.UpdateRequirement(requirement)
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

func (bx *BuildExpression) AddRequirement(requirement Requirement) error {
	bx.requirements = append(bx.requirements, requirement)

	(*bx.solveNode)[RequirementsKey] = bx.requirements

	return nil
}

func (bx *BuildExpression) RemoveRequirement(requirement Requirement) error {
	for i, req := range bx.requirements {
		if req.Name == requirement.Name && req.Namespace == requirement.Namespace {
			bx.requirements = append(bx.requirements[:i], bx.requirements[i+1:]...)
			(*bx.solveNode)[RequirementsKey] = bx.requirements
			return nil
		}
	}

	return errs.New("Could not find requirement")
}

func (bx BuildExpression) UpdateRequirement(requirement Requirement) error {
	for i, req := range bx.requirements {
		if req.Name == requirement.Name && req.Namespace == requirement.Namespace {
			bx.requirements[i] = requirement
			(*bx.solveNode)[RequirementsKey] = bx.requirements
			return nil
		}
	}

	return errs.New("Could not find requirement")
}

func (bx BuildExpression) UpdateTimestamp() error {
	(*bx.solveNode)[AtTimeKey] = time.Now().UTC().Format(time.RFC3339)
	return nil
}

func getRequirementsNode(bx map[string]interface{}) ([]interface{}, error) {
	solveNode, err := getSolveNode(bx)
	if err != nil {
		return nil, errs.Wrap(err, "Could not get solve node")
	}

	for k, v := range solveNode {
		if k != RequirementsKey {
			continue
		}

		node, ok := v.([]interface{})
		if !ok {
			return nil, errs.New("Requirements in BuildExpression are malformed")
		}

		return node, nil
	}

	return nil, errs.New("Could not find requirements node")
}

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

func getFuncNode(expression map[string]interface{}, funcName string) (map[string]interface{}, error) {
	for k, v := range expression {
		node, ok := v.(map[string]interface{})
		if !ok {
			continue
		}

		if k == funcName {
			return node, nil
		}

		if childNode, err := getFuncNode(node, funcName); err == nil {
			return childNode, nil
		}
	}

	return nil, funcNodeNotFoundError
}
