package merge

import (
	"encoding/json"
	"reflect"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/multilog"
	bpModel "github.com/ActiveState/cli/pkg/platform/api/buildplanner/model"
	"github.com/ActiveState/cli/pkg/platform/api/mono/mono_models"
	"github.com/ActiveState/cli/pkg/platform/runtime/buildexpression"
)

type AutoMergeNotPossibleError struct{ error }

func IsAutoMergeNotPossibleError(err error) bool {
	return errs.Matches(err, &AutoMergeNotPossibleError{})
}

type MergeConflictsError struct{ error }

func IsMergeConflictsError(err error) bool {
	return errs.Matches(err, &MergeConflictsError{})
}

func Merge(exprA *buildexpression.BuildExpression, exprB *buildexpression.BuildExpression, strategies *mono_models.MergeStrategies) (*buildexpression.BuildExpression, error) {
	if !isAutoMergePossible(exprA, exprB) {
		return nil, &AutoMergeNotPossibleError{errs.New("Unable to merge buildexpressions")}
	}
	if len(strategies.Conflicts) > 0 {
		return nil, &MergeConflictsError{errs.New("Unable to merge buildexpressions due to conflicting requirements")}
	}

	// Update build expression requirements with merge results.
	for _, req := range strategies.OverwriteChanges {
		var op bpModel.Operation
		switch req.Operation {
		case mono_models.CommitChangeEditableOperationAdded:
			op = bpModel.OperationAdded
		case mono_models.CommitChangeEditableOperationRemoved:
			op = bpModel.OperationRemoved
		case mono_models.CommitChangeEditableOperationUpdated:
			op = bpModel.OperationUpdated
		default:
			return nil, errs.New("Unknown requirement operation: %s", op)
		}

		var versionRequirements []bpModel.VersionRequirement
		for _, constraint := range req.VersionConstraints {
			data, err := constraint.MarshalBinary()
			if err != nil {
				return nil, errs.Wrap(err, "Could not marshal requirement version constraints")
			}
			m := make(map[string]string)
			err = json.Unmarshal(data, &m)
			if err != nil {
				return nil, errs.Wrap(err, "Could not unmarshal requirement version constraints")
			}
			versionRequirements = append(versionRequirements, m)
		}

		bpReq := bpModel.Requirement{
			Name:               req.Requirement,
			Namespace:          req.Namespace,
			VersionRequirement: versionRequirements,
		}
		exprB.Update(op, bpReq, nil)
	}

	return exprB, nil
}

// isAutoMergePossible determines whether or not it is possible to auto-merge the given build
// expressions.
// This is only possible if the two build expressions differ ONLY in requirements.
func isAutoMergePossible(exprA *buildexpression.BuildExpression, exprB *buildexpression.BuildExpression) bool {
	jsonA, err := getJsonMinusRequirements(exprA)
	if err != nil {
		multilog.Error("Unable to get buildexpression minus requirements: %v", errs.JoinMessage(err))
		return false
	}
	jsonB, err := getJsonMinusRequirements(exprB)
	if err != nil {
		multilog.Error("Unable to get buildxpression minus requirements: %v", errs.JoinMessage(err))
		return false
	}
	return reflect.DeepEqual(jsonA, jsonB)
}

// getJsonMinusRequirements returns a JSON map[string]interface{} structure for the given build
// expression without a "requirements" key nested inside the build expression.
func getJsonMinusRequirements(expr *buildexpression.BuildExpression) (map[string]interface{}, error) {
	data, err := json.Marshal(expr)
	if err != nil {
		return nil, errs.New("Unable to unmarshal marshaled buildxpression")
	}

	m := make(map[string]interface{})
	err = json.Unmarshal(data, &m)
	if err != nil {
		return nil, errs.New("Unable to unmarshal marshaled buildxpression")
	}

	letValue, ok := m["let"]
	if !ok {
		return nil, errs.New("Build expression has no 'let' key")
	}
	letMap, ok := letValue.(map[string]interface{})
	if !ok {
		return nil, errs.New("'let' key is not a JSON object")
	}
	deleteRequirements(&letMap)

	return m, nil
}

// deleteRequirements recursively iterates over the given JSON map until it finds a "requirements"
// key and deletes it and its value.
func deleteRequirements(m *map[string]interface{}) bool {
	for k, v := range *m {
		if k == "requirements" {
			delete(*m, k)
			return true
		}
		if m2, ok := v.(map[string]interface{}); ok {
			if deleteRequirements(&m2) {
				return true
			}
		}
	}
	return false
}
