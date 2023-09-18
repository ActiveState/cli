package merge

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

	strategies := &mono_models.MergeStrategies{
		OverwriteChanges: []*mono_models.CommitChangeEditable{
			{Namespace: "language/perl", Requirement: "DateTime", Operation: mono_models.CommitChangeEditableOperationAdded},
		},
	}

	require.True(t, isAutoMergePossible(exprA, exprB))

	mergedExpr, err := Merge(exprA, exprB, strategies)
	require.NoError(t, err)

	mergedScript, err := buildscript.NewScriptFromBuildExpression(mergedExpr)
	require.NoError(t, err)

	// TODO: delete this block after DX-1939. Sorting requirements is needed until we have
	// buildexpression hashes for comparing equality.
	assert.Equal(t,
		`let:
	runtime = solve(
		platforms = [
			"12345",
			"67890"
		],
		requirements = [
			{
				name = "JSON",
				namespace = "language/perl"
			},
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
	return

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

	// Note the intentional swap of platform order. Buildexpression list order does not matter.
	// isAutoMergePossible() should still return true, and the original platforms will be used.
	scriptB, err := buildscript.NewScript([]byte(
		`let:
	runtime = solve(
		platforms = [
			"67890",
			"12345"
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

	strategies := &mono_models.MergeStrategies{
		OverwriteChanges: []*mono_models.CommitChangeEditable{
			{Namespace: "language/perl", Requirement: "JSON", Operation: mono_models.CommitChangeEditableOperationRemoved},
		},
	}

	require.True(t, isAutoMergePossible(exprA, exprB))

	mergedExpr, err := Merge(exprA, exprB, strategies)
	require.NoError(t, err)

	mergedScript, err := buildscript.NewScriptFromBuildExpression(mergedExpr)
	require.NoError(t, err)

	// TODO: delete this block after DX-1939. Sorting requirements is needed until we have
	// buildexpression hashes for comparing equality.
	assert.Equal(t,
		`let:
	runtime = solve(
		platforms = [
			"12345",
			"67890"
		],
		requirements = [
			{
				name = "DateTime",
				namespace = "language/perl"
			},
			{
				name = "perl",
				namespace = "language"
			}
		]
	)

in:
	runtime`, mergedScript.String())
	return

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

	_, err = Merge(exprA, exprB, nil)
	require.Error(t, err)
}

func TestDeleteKey(t *testing.T) {
	m := map[string]interface{}{"foo": map[string]interface{}{"bar": "baz", "quux": "foobar"}}
	assert.True(t, deleteKey(&m, "quux"), "did not find quux")
	_, exists := m["foo"].(map[string]interface{})["quux"]
	assert.False(t, exists, "did not delete quux")
}

func TestSortLists(t *testing.T) {
	m := map[string]interface{}{
		"one": []interface{}{"foo", "bar", "baz"},
		"two": map[string]interface{}{
			"three": []interface{}{"foobar", "barfoo", "barbaz"},
		},
	}
	sortLists(&m)
	assert.Equal(t, []interface{}{"bar", "baz", "foo"}, m["one"])
	assert.Equal(t, []interface{}{"barbaz", "barfoo", "foobar"}, m["two"].(map[string]interface{})["three"])
}
