package buildscript

import (
	"encoding/json"
	"testing"

	"github.com/ActiveState/cli/pkg/platform/api/mono/mono_models"
	"github.com/ActiveState/cli/pkg/platform/runtime/buildexpression"
	"github.com/ActiveState/cli/pkg/platform/runtime/buildscript"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMergeAdd(t *testing.T) {
	scriptA, err := buildscript.NewScript([]byte(
		`let:
	runtime = solve(
		platforms = [
			"12345",
			"67890"
		],
		requirements = [
			{
				name = "perl",
				namespace = "language"
			},
			{
				name = "DateTime",
				namespace = "language/perl"
			}
		]
	)

in:
	runtime`))
	require.NoError(t, err)
	bytes, err := json.Marshal(scriptA)
	require.NoError(t, err)
	exprA, err := buildexpression.New(bytes)
	require.NoError(t, err)

	scriptB, err := buildscript.NewScript([]byte(
		`let:
	runtime = solve(
		platforms = [
			"12345",
			"67890"
		],
		requirements = [
			{
				name = "perl",
				namespace = "language"
			},
			{
				name = "JSON",
				namespace = "language/perl"
			}
		]
	)

in:
	runtime`))
	require.NoError(t, err)
	bytes, err = json.Marshal(scriptB)
	require.NoError(t, err)
	exprB, err := buildexpression.New(bytes)
	require.NoError(t, err)

	require.True(t, isAutoMergePossible(exprA, exprB))

	strategies := &mono_models.MergeStrategies{
		OverwriteChanges: []*mono_models.CommitChangeEditable{
			{Namespace: "language/perl", Requirement: "DateTime", Operation: mono_models.CommitChangeEditableOperationAdded},
		},
	}

	mergedExpr, err := apply(exprB, strategies)
	require.NoError(t, err)

	bytes, err = json.Marshal(mergedExpr)
	require.NoError(t, err)
	mergedScript, err := buildscript.NewScriptFromBuildExpression(bytes)
	require.NoError(t, err)

	assert.Equal(t,
		`let:
	runtime = solve(
		platforms = [
			"12345",
			"67890"
		],
		requirements = [
			{
				name = "perl",
				namespace = "language"
			},
			{
				name = "JSON",
				namespace = "language/perl"
			},
			{
				name = "DateTime",
				namespace = "language/perl"
			}
		]
	)

in:
	runtime`, mergedScript.String())
}

func TestMergeRemove(t *testing.T) {
	scriptA, err := buildscript.NewScript([]byte(
		`let:
	runtime = solve(
		platforms = [
			"12345",
			"67890"
		],
		requirements = [
			{
				name = "perl",
				namespace = "language"
			},
			{
				name = "JSON",
				namespace = "language/perl"
			},
			{
				name = "DateTime",
				namespace = "language/perl"
			}
		]
	)

in:
	runtime`))
	require.NoError(t, err)
	bytes, err := json.Marshal(scriptA)
	require.NoError(t, err)
	exprA, err := buildexpression.New(bytes)
	require.NoError(t, err)

	scriptB, err := buildscript.NewScript([]byte(
		`let:
	runtime = solve(
		platforms = [
			"12345",
			"67890"
		],
		requirements = [
			{
				name = "perl",
				namespace = "language"
			},
			{
				name = "DateTime",
				namespace = "language/perl"
			}
		]
	)

in:
	runtime`))
	require.NoError(t, err)
	bytes, err = json.Marshal(scriptB)
	require.NoError(t, err)
	exprB, err := buildexpression.New(bytes)
	require.NoError(t, err)

	require.True(t, isAutoMergePossible(exprA, exprB))

	strategies := &mono_models.MergeStrategies{
		OverwriteChanges: []*mono_models.CommitChangeEditable{
			{Namespace: "language/perl", Requirement: "JSON", Operation: mono_models.CommitChangeEditableOperationRemoved},
		},
	}

	mergedExpr, err := apply(exprB, strategies)
	require.NoError(t, err)

	bytes, err = json.Marshal(mergedExpr)
	require.NoError(t, err)
	mergedScript, err := buildscript.NewScriptFromBuildExpression(bytes)
	require.NoError(t, err)

	assert.Equal(t,
		`let:
	runtime = solve(
		platforms = [
			"12345",
			"67890"
		],
		requirements = [
			{
				name = "perl",
				namespace = "language"
			},
			{
				name = "DateTime",
				namespace = "language/perl"
			}
		]
	)

in:
	runtime`, mergedScript.String())
}

func TestMergeConflict(t *testing.T) {
	scriptA, err := buildscript.NewScript([]byte(
		`let:
	runtime = solve(
		platforms = [
			"12345",
			"67890"
		],
		requirements = [
			{
				name = "perl",
				namespace = "language"
			},
			{
				name = "DateTime",
				namespace = "language/perl"
			}
		]
	)

in:
	runtime`))
	require.NoError(t, err)
	bytes, err := json.Marshal(scriptA)
	require.NoError(t, err)
	exprA, err := buildexpression.New(bytes)
	require.NoError(t, err)

	scriptB, err := buildscript.NewScript([]byte(
		`let:
	runtime = solve(
		platforms = [
			"12345"
		],
		requirements = [
			{
				name = "perl",
				namespace = "language"
			},
			{
				name = "JSON",
				namespace = "language/perl"
			}
		]
	)

in:
	runtime`))
	require.NoError(t, err)
	bytes, err = json.Marshal(scriptB)
	require.NoError(t, err)
	exprB, err := buildexpression.New(bytes)
	require.NoError(t, err)

	assert.False(t, isAutoMergePossible(exprA, exprB)) // platforms do not match
}
