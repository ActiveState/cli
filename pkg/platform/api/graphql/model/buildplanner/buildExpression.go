package model

import (
	"encoding/json"
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

	SolveFuncName       = "solve"
	SolveLegacyFuncName = "solve_legacy"
	RequirementsKey     = "requirements"
	AtTimeKey           = "at_time"
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

	logging.Debug("BuildExpresison Data: %s", data)

	return bx, nil
}

func (bx BuildExpression) Requirements() ([]Requirement, error) {
	solveLegacyNode, err := getFuncNode(bx, SolveLegacyFuncName)
	if err != nil {
		return nil, errs.Wrap(err, "Could not get solve legacy node")
	}

	requirementsNode, err := getRequirementsNode(solveLegacyNode)
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
		return errs.Wrap(err, "Could not update BuildExpression's requirement")
	}

	err = bx.UpdateTimestamp()
	if err != nil {
		return errs.Wrap(err, "Could not update BuildExpression timestamp")
	}

	return nil
}

func (bx BuildExpression) UpdateRequirements(requirements []Requirement) error {
	solveLegacyNode, err := getFuncNode(bx, SolveLegacyFuncName)
	if err != nil {
		return errs.Wrap(err, "Could not get solve legacy node")
	}

	requirementsData, err := json.Marshal(requirements)
	if err != nil {
		return errs.Wrap(err, "Could not marshal JSON")
	}

	var newRequirements []interface{}
	err = json.Unmarshal(requirementsData, &newRequirements)
	if err != nil {
		return errs.Wrap(err, "Could not unmarshal JSON")
	}

	solveLegacyNode[RequirementsKey] = newRequirements

	return nil
}

func (bx BuildExpression) AddRequirement(requirement Requirement) error {
	solveLegacyNode, err := getFuncNode(bx, SolveLegacyFuncName)
	if err != nil {
		return errs.Wrap(err, "Could not get solve legacy node")
	}

	requirementsNode, err := getRequirementsNode(solveLegacyNode)
	if err != nil {
		return errs.Wrap(err, "Could not get requirements node")
	}

	newRequirementData, err := json.Marshal(requirement)
	if err != nil {
		return errs.Wrap(err, "Could not marshal JSON")
	}

	var newRequirement interface{}
	err = json.Unmarshal(newRequirementData, &newRequirement)
	if err != nil {
		return errs.Wrap(err, "Could not unmarshal JSON")
	}

	requirementsNode = append(requirementsNode, newRequirement)

	solveLegacyNode[RequirementsKey] = requirementsNode

	return nil
}

func (bx BuildExpression) RemoveRequirement(requirement Requirement) error {
	solveLegacyNode, err := getFuncNode(bx, SolveLegacyFuncName)
	if err != nil {
		return errs.Wrap(err, "Could not get solve legacy node")
	}

	requirementsNode, err := getRequirementsNode(solveLegacyNode)
	if err != nil {
		return errs.Wrap(err, "Could not get requirements node")
	}

	for i, req := range requirementsNode {
		r, ok := req.(map[string]interface{})
		if !ok {
			return errs.New("Requirement in BuildExpression is malformed, type: %T", req)
		}

		if r["name"] == requirement.Name && r["namespace"] == requirement.Namespace {
			requirementsNode = append(requirementsNode[:i], requirementsNode[i+1:]...)
			solveLegacyNode[RequirementsKey] = requirementsNode
			return nil
		}
	}

	return errs.New("Could not find requirement")
}

func (bx BuildExpression) UpdateRequirement(requirement Requirement) error {
	solveLegacyNode, err := getFuncNode(bx, SolveLegacyFuncName)
	if err != nil {
		return errs.Wrap(err, "Could not get solve legacy node")
	}

	requirementsNode, err := getRequirementsNode(solveLegacyNode)
	if err != nil {
		return errs.Wrap(err, "Could not get requirements node")
	}

	for i, req := range requirementsNode {
		r, ok := req.(map[string]interface{})
		if !ok {
			return errs.New("Requirement in BuildExpression is malformed")
		}

		if r["name"] == requirement.Name && r["namespace"] == requirement.Namespace {
			requirementsNode[i] = requirement
			solveLegacyNode[RequirementsKey] = requirementsNode
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

func getFuncNode(bx BuildExpression, funcName string) (map[string]interface{}, error) {
	for k, v := range bx {
		node, ok := v.(map[string]interface{})
		if !ok {
			continue
		}

		if k == funcName {
			return node, nil
		}

		return getFuncNode(node, funcName)
	}

	return nil, errs.New("Could not find solve node")
}

func getRequirementsNode(solveNode map[string]interface{}) ([]interface{}, error) {
	for k, v := range solveNode {
		if k != RequirementsKey {
			continue
		}

		node, ok := v.([]interface{})
		if !ok {
			return nil, errs.New("Requirements key in JSON object is malformed")
		}

		return node, nil
	}

	return nil, errs.New("Could not find requirements node")
}
