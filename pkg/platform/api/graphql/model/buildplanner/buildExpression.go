package model

import (
	"encoding/json"
	"errors"
	"time"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/logging"
)

type Comparator string

type Operation int

const (
	ComparatorEQ  Comparator = "eq"
	ComparatorGT             = "gt"
	ComparatorGTE            = "gte"
	ComparatorLT             = "lt"
	ComparatorLTE            = "lte"
	ComparatorNE             = "ne"

	OperationAdd Operation = iota
	OperationRemove
	OperationUpdate

	SolveFuncName           = "solve"
	SolveLegacyFuncName     = "solve_legacy"
	RequirementsKey         = "requirements"
	AtTimeKey               = "at_time"
	RequirementNameKey      = "name"
	RequirementNamespaceKey = "namespace"
)

func (o Operation) String() string {
	switch o {
	case OperationAdd:
		return "add"
	case OperationRemove:
		return "remove"
	case OperationUpdate:
		return "update"
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

type VersionRequirement map[Comparator]string

type BuildExpression map[string]interface{}

func NewBuildExpression(data []byte) (BuildExpression, error) {
	var bx BuildExpression

	err := json.Unmarshal(data, &bx)
	if err != nil {
		return nil, errs.Wrap(err, "Could not unmarshal JSON")
	}

	return bx, nil
}

func (bx BuildExpression) Requirements() ([]Requirement, error) {
	requirementsNode, err := getRequirementsNode(bx)
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

	return requirements, nil
}

func (bx BuildExpression) Update(operation Operation, requirement Requirement) error {
	var err error
	switch operation {
	case OperationAdd:
		err = bx.AddRequirement(requirement)
	case OperationRemove:
		err = bx.RemoveRequirement(requirement)
	case OperationUpdate:
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

func (bx BuildExpression) AddRequirement(requirement Requirement) error {
	solveNode, err := getSolveNode(bx)
	if err != nil {
		return errs.Wrap(err, "Could not get solve node")
	}

	requirementsNode, err := getRequirementsNode(bx)
	if err != nil {
		return errs.Wrap(err, "Could not get requirements node")
	}

	newRequirementData, err := json.Marshal(requirement)
	if err != nil {
		return errs.Wrap(err, "Could not marshal JSON")
	}

	var newRequirement Requirement
	err = json.Unmarshal(newRequirementData, &newRequirement)
	if err != nil {
		return errs.Wrap(err, "Could not unmarshal JSON")
	}

	requirementsNode = append(requirementsNode, newRequirement)

	solveNode[RequirementsKey] = requirementsNode

	return nil
}

func (bx BuildExpression) RemoveRequirement(requirement Requirement) error {
	solveNode, err := getSolveNode(bx)
	if err != nil {
		return errs.Wrap(err, "Could not get solve node")
	}

	requirementsNode, err := getRequirementsNode(bx)
	if err != nil {
		return errs.Wrap(err, "Could not get requirements node")
	}

	for i, req := range requirementsNode {
		r, ok := req.(map[string]interface{})
		if !ok {
			return errs.New("Requirement in BuildExpression is malformed")
		}

		if r[RequirementNameKey] == requirement.Name && r[RequirementNamespaceKey] == requirement.Namespace {
			requirementsNode = append(requirementsNode[:i], requirementsNode[i+1:]...)
			solveNode[RequirementsKey] = requirementsNode
			return nil
		}
	}

	return errs.New("Could not find requirement")
}

func (bx BuildExpression) UpdateRequirement(requirement Requirement) error {
	solveNode, err := getSolveNode(bx)
	if err != nil {
		return errs.Wrap(err, "Could not get solve legacy node")
	}

	requirementsNode, err := getRequirementsNode(bx)
	if err != nil {
		return errs.Wrap(err, "Could not get requirements node")
	}

	for i, req := range requirementsNode {
		r, ok := req.(map[string]interface{})
		if !ok {
			return errs.New("Requirement in BuildExpression is malformed")
		}

		if r[RequirementNameKey] == requirement.Name && r[RequirementNamespaceKey] == requirement.Namespace {
			requirementsNode[i] = requirement
			solveNode[RequirementsKey] = requirementsNode
			return nil
		}
	}

	return errs.New("Could not find requirement")
}

func (bx BuildExpression) UpdateTimestamp() error {
	solveLegacyNode, err := getFuncNode(bx, SolveLegacyFuncName)
	if err != nil {
		return errs.Wrap(err, "Could not get solve legacy node")
	}

	solveLegacyNode[AtTimeKey] = time.Now().UTC().Format(time.RFC3339)

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

func getSolveNode(bx BuildExpression) (map[string]interface{}, error) {
	solveNode, err := getFuncNode(bx, SolveFuncName)
	if err == nil {
		return solveNode, nil
	}
	if !errors.Is(err, funcNodeNotFoundError) {
		return nil, errs.Wrap(err, "Could not get solve node")
	}
	logging.Debug("Could not get solve node, trying solve legacy node")

	solveNode, err = getFuncNode(bx, SolveLegacyFuncName)
	if err != nil {
		return nil, errs.Wrap(err, "Could not get solve legacy node")
	}

	return solveNode, nil
}

func getFuncNode(bx BuildExpression, funcName string) (map[string]interface{}, error) {
	for k, v := range bx {
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
