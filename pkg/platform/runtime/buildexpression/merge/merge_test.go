package merge

import (
	"testing"

	"github.com/ActiveState/cli/pkg/platform/api/mono/mono_models"
	"github.com/ActiveState/cli/pkg/platform/runtime/buildscript"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMergeAdd(t *testing.T) {
	scriptA, err := buildscript.New([]byte(
		`at_time = "2000-01-01T00:00:00.000Z"
runtime = solve(
	at_time = at_time,
	platforms = [
		"12345",
		"67890"
	],
	requirements = [
		Req(name = "language/perl"),
		Req(name = "language/perl/DateTime")
	]
)

main = runtime`))
	require.NoError(t, err)
	exprA := scriptA.Expr

	scriptB, err := buildscript.New([]byte(
		`at_time = "2000-01-01T00:00:00.000Z"
runtime = solve(
	at_time = at_time,
	platforms = [
		"12345",
		"67890"
	],
	requirements = [
		Req(name = "language/perl"),
		Req(name = "language/perl/JSON")
	]
)

main = runtime`))
	require.NoError(t, err)
	exprB := scriptB.Expr

	strategies := &mono_models.MergeStrategies{
		OverwriteChanges: []*mono_models.CommitChangeEditable{
			{Namespace: "language/perl", Requirement: "DateTime", Operation: mono_models.CommitChangeEditableOperationAdded},
		},
	}

	require.True(t, isAutoMergePossible(exprA, exprB))

	mergedExpr, err := Merge(exprA, exprB, strategies)
	require.NoError(t, err)

	mergedScript, err := buildscript.NewFromCommit(scriptA.AtTime, mergedExpr)
	require.NoError(t, err)

	assert.Equal(t,
		`at_time = "2000-01-01T00:00:00.000Z"
runtime = solve(
	at_time = at_time,
	platforms = [
		"12345",
		"67890"
	],
	requirements = [
		Req(name = "language/perl"),
		Req(name = "language/perl/JSON"),
		Req(name = "language/perl/DateTime")
	]
)

main = runtime`, mergedScript.String())
}

func TestMergeRemove(t *testing.T) {
	scriptA, err := buildscript.New([]byte(
		`at_time = "2000-01-01T00:00:00.000Z"
runtime = solve(
	at_time = at_time,
	platforms = [
		"12345",
		"67890"
	],
	requirements = [
		Req(name = "language/perl"),
		Req(name = "language/perl/JSON"),
		Req(name = "language/perl/DateTime")
	]
)

main = runtime`))
	require.NoError(t, err)
	exprA := scriptA.Expr

	scriptB, err := buildscript.New([]byte(
		`at_time = "2000-01-01T00:00:00.000Z"
runtime = solve(
	at_time = at_time,
	platforms = [
		"12345",
		"67890"
	],
	requirements = [
		Req(name = "language/perl"),
		Req(name = "language/perl/DateTime")
	]
)

main = runtime`))
	require.NoError(t, err)
	exprB := scriptB.Expr

	strategies := &mono_models.MergeStrategies{
		OverwriteChanges: []*mono_models.CommitChangeEditable{
			{Namespace: "language/perl", Requirement: "JSON", Operation: mono_models.CommitChangeEditableOperationRemoved},
		},
	}

	require.True(t, isAutoMergePossible(exprA, exprB))

	mergedExpr, err := Merge(exprA, exprB, strategies)
	require.NoError(t, err)

	mergedScript, err := buildscript.NewFromCommit(scriptA.AtTime, mergedExpr)
	require.NoError(t, err)

	assert.Equal(t,
		`at_time = "2000-01-01T00:00:00.000Z"
runtime = solve(
	at_time = at_time,
	platforms = [
		"12345",
		"67890"
	],
	requirements = [
		Req(name = "language/perl"),
		Req(name = "language/perl/DateTime")
	]
)

main = runtime`, mergedScript.String())
}

func TestMergeConflict(t *testing.T) {
	scriptA, err := buildscript.New([]byte(
		`at_time = "2000-01-01T00:00:00.000Z"
runtime = solve(
	at_time = at_time,
	platforms = [
		"12345",
		"67890"
	],
	requirements = [
		Req(name = "language/perl"),
	]
)

main = runtime`))
	require.NoError(t, err)
	exprA := scriptA.Expr

	scriptB, err := buildscript.New([]byte(
		`at_time = "2000-01-01T00:00:00.000Z"
runtime = solve(
	at_time = at_time,
	platforms = [
		"12345"
	],
	requirements = [
		Req(name = "language/perl"),
		Req(name = "language/perl/JSON")
	]
)

main = runtime`))
	require.NoError(t, err)
	exprB := scriptB.Expr

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
