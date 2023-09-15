package merge

import (
	"encoding/json"
	"reflect"
	"sort"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/multilog"
	bpModel "github.com/ActiveState/cli/pkg/platform/api/buildplanner/model"
	"github.com/ActiveState/cli/pkg/platform/api/mono/mono_models"
	"github.com/ActiveState/cli/pkg/platform/runtime/buildexpression"
)

func Merge(exprA *buildexpression.BuildExpression, exprB *buildexpression.BuildExpression, strategies *mono_models.MergeStrategies) (*buildexpression.BuildExpression, error) {
	if !isAutoMergePossible(exprA, exprB) {
		return nil, errs.New("Unable to merge buildexpressions")
	}
	if len(strategies.Conflicts) > 0 {
		return nil, errs.New("Unable to merge buildexpressions due to conflicting requirements")
	}

	// Update build expression requirements with merge results.
	for _, req := range strategies.OverwriteChanges {
		var op bpModel.Operation
		err := op.Unmarshal(req.Operation)
		if err != nil {
			return nil, errs.Wrap(err, "Unable to convert requirement operation to buildplan operation")
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
		exprB.UpdateRequirement(op, bpReq)
	}

	return exprB, nil
}

// isAutoMergePossible determines whether or not it is possible to auto-merge the given build
// expressions.
// This is only possible if the two build expressions differ ONLY in requirements.
func isAutoMergePossible(exprA *buildexpression.BuildExpression, exprB *buildexpression.BuildExpression) bool {
	jsonA, err := getComparableJson(exprA)
	if err != nil {
		multilog.Error("Unable to get buildexpression minus requirements: %v", errs.JoinMessage(err))
		return false
	}
	jsonB, err := getComparableJson(exprB)
	if err != nil {
		multilog.Error("Unable to get buildxpression minus requirements: %v", errs.JoinMessage(err))
		return false
	}
	logging.Debug("Checking for possibility of auto-merging build expressions")
	logging.Debug("JsonA: %v", jsonA)
	logging.Debug("JsonB: %v", jsonB)
	return reflect.DeepEqual(jsonA, jsonB) // TODO: replace with DX-1939 solution
}

// getComparableJson returns a comparable JSON map[string]interface{} structure for the given build
// expression. The map will not have a "requirements" field, nor will it have an "at_time" field.
// String lists will also be sorted.
func getComparableJson(expr *buildexpression.BuildExpression) (map[string]interface{}, error) {
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
	deleteKey(&letMap, "requirements")
	// TODO: the following shouldn't be needed after DX-1939.
	sortLists(&letMap)
	deleteKey(&letMap, "at_time")

	return m, nil
}

// deleteKey recursively iterates over the given JSON map until it finds the given key and deletes
// it and its value.
func deleteKey(m *map[string]interface{}, key string) bool {
	for k, v := range *m {
		if k == key {
			delete(*m, k)
			return true
		}
		if m2, ok := v.(map[string]interface{}); ok {
			if deleteKey(&m2, key) {
				return true
			}
		}
	}
	return false
}

// sortLists recursively iterates over the given JSON map looking for string lists, and sorts them.
// This is needed because isAutoMergePossible() does a reflect.DeepEqual(), but build expression
// list order does not matter.
// This will not be necessary after DX-1939 is implemented.
func sortLists(m *map[string]interface{}) {
	for _, v := range *m {
		if list, ok := v.([]interface{}); ok {
			sort.SliceStable(list, func(i, j int) bool {
				s1, ok1 := list[i].(string)
				s2, ok2 := list[j].(string)
				return ok1 && ok2 && s1 < s2
			})
		} else if m2, ok := v.(map[string]interface{}); ok {
			sortLists(&m2)
		}
	}
}
