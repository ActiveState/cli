package buildscript

import (
	"encoding/json"
	"reflect"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/multilog"
	"github.com/ActiveState/cli/pkg/platform/api/buildplanner/types"
	"github.com/ActiveState/cli/pkg/platform/api/mono/mono_models"
)

// Merge merges the requirements from another BuildScript into this one, according to the given
// merge strategy.
// BuildScript merges are only possible if the scripts differ ONLY in requirements AND/OR at times.
func (b *BuildScript) Merge(other *BuildScript, strategies *mono_models.MergeStrategies) error {
	if !isAutoMergePossible(b, other) {
		return errs.New("Unable to merge build scripts")
	}
	if len(strategies.Conflicts) > 0 {
		return errs.New("Unable to merge build scripts due to conflicting requirements")
	}

	// Update requirements with merge results.
	for _, req := range strategies.OverwriteChanges {
		var op types.Operation
		err := op.Unmarshal(req.Operation)
		if err != nil {
			return errs.Wrap(err, "Unable to convert requirement operation to buildplan operation")
		}

		var versionRequirements []types.VersionRequirement
		for _, constraint := range req.VersionConstraints {
			data, err := constraint.MarshalBinary()
			if err != nil {
				return errs.Wrap(err, "Could not marshal requirement version constraints")
			}
			m := make(map[string]string)
			err = json.Unmarshal(data, &m)
			if err != nil {
				return errs.Wrap(err, "Could not unmarshal requirement version constraints")
			}
			versionRequirements = append(versionRequirements, m)
		}

		bpReq := types.Requirement{
			Name:               req.Requirement,
			Namespace:          req.Namespace,
			VersionRequirement: versionRequirements,
		}

		if err := b.UpdateRequirement(op, bpReq); err != nil {
			return errs.Wrap(err, "Unable to update build script with merge results")
		}
	}

	// When merging build scripts we want to use the most recent timestamp
	atTime := other.AtTime()
	if atTime != nil && atTime.After(*b.AtTime()) {
		b.SetAtTime(*atTime)
	}

	return nil
}

// isAutoMergePossible determines whether or not it is possible to auto-merge the given build
// scripts.
// This is only possible if the two build scripts differ ONLY in requirements.
func isAutoMergePossible(scriptA *BuildScript, scriptB *BuildScript) bool {
	jsonA, err := getComparableJson(scriptA)
	if err != nil {
		multilog.Error("Unable to get build script minus requirements: %v", errs.JoinMessage(err))
		return false
	}
	jsonB, err := getComparableJson(scriptB)
	if err != nil {
		multilog.Error("Unable to get build script minus requirements: %v", errs.JoinMessage(err))
		return false
	}
	logging.Debug("Checking for possibility of auto-merging build scripts")
	logging.Debug("JsonA: %v", jsonA)
	logging.Debug("JsonB: %v", jsonB)
	return reflect.DeepEqual(jsonA, jsonB)
}

// getComparableJson returns a comparable JSON map[string]interface{} structure for the given build
// script. The map will not have a "requirements" field.
func getComparableJson(script *BuildScript) (map[string]interface{}, error) {
	data, err := script.MarshalBuildExpression()
	if err != nil {
		return nil, errs.New("Unable to unmarshal marshaled build expression")
	}

	m := make(map[string]interface{})
	err = json.Unmarshal(data, &m)
	if err != nil {
		return nil, errs.New("Unable to unmarshal marshaled build expression")
	}

	letMap, ok := m["let"].(map[string]interface{})
	if !ok {
		return nil, errs.New("'let' key is not a JSON object")
	}
	deleteKey(&letMap, "requirements")

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
