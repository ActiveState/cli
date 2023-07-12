package buildscript

import (
	"encoding/json"
	"reflect"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/multilog"
	"github.com/ActiveState/cli/pkg/localcommit"
	bpModel "github.com/ActiveState/cli/pkg/platform/api/buildplanner/model"
	"github.com/ActiveState/cli/pkg/platform/api/mono/mono_models"
	"github.com/ActiveState/cli/pkg/platform/authentication"
	"github.com/ActiveState/cli/pkg/platform/model"
	"github.com/ActiveState/cli/pkg/platform/runtime/buildexpression"
	"github.com/ActiveState/cli/pkg/platform/runtime/buildscript"
	"github.com/ActiveState/cli/pkg/project"
	"github.com/go-openapi/strfmt"
)

func Merge(proj *project.Project, remoteCommit strfmt.UUID, strategies *mono_models.MergeStrategies, auth *authentication.Auth) error {
	// Verify we have a build script to merge.
	script, err := buildscript.NewScriptFromProjectDir(proj.Dir())
	if err != nil && !buildscript.IsDoesNotExistError(err) {
		return errs.Wrap(err, "Could not get local build script")
	}
	if script == nil {
		return nil // no build script to merge
	}

	// Get the local and remote build expressions to check for mergeability.
	bp := model.NewBuildPlannerModel(auth)
	localCommit, err := localcommit.Get(proj.Dir())
	if err != nil {
		return errs.Wrap(err, "Unable to get local commit")
	}
	exprA, err := bp.GetBuildExpression(proj.Owner(), proj.Name(), localCommit.String())
	if err != nil {
		return errs.Wrap(err, "Unable to get buildexpression for local commit")
	}
	exprB, err := bp.GetBuildExpression(proj.Owner(), proj.Name(), remoteCommit.String())
	if err != nil {
		return errs.Wrap(err, "Unable to get buildexpression for remote commit")
	}

	// Check if the build expressions can be auto-merged.
	if !isAutoMergePossible(exprA, exprB) || len(strategies.Conflicts) > 0 {
		err := GenerateAndWriteDiff(proj, script, exprB)
		if err != nil {
			return locale.WrapError(err, "err_diff_build_script", "Unable to generate differences between local and remote build script")
		}
		return locale.NewInputError("err_build_script_merge", "Unable to automatically merge build scripts")
	}

	mergedExpr, err := apply(exprB, strategies)
	if err != nil {
		return errs.Wrap(err, "Unable to merge buildexpressions")
	}

	// Write the merged build expression as a local build script.
	return buildscript.UpdateOrCreate(proj.Dir(), mergedExpr)
}

func apply(expr *buildexpression.BuildExpression, strategies *mono_models.MergeStrategies) (*buildexpression.BuildExpression, error) {
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
		expr.Update(op, bpReq, nil)
	}

	return expr, nil
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
