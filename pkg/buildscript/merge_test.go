package buildscript

import (
	"testing"

	"github.com/ActiveState/cli/pkg/platform/api/mono/mono_models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const mergeATime = "2000-01-01T00:00:00.000Z"
const mergeBTime = "2000-01-02T00:00:00.000Z"

func TestMergeAdd(t *testing.T) {
	scriptA, err := Unmarshal([]byte(
		checkoutInfo(testProject, mergeATime) + `
runtime = solve(
	at_time = at_time,
	platforms = [
		"12345",
		"67890"
	],
	requirements = [
		Req(name = "perl", namespace = "language"),
		Req(name = "DateTime", namespace = "language/perl")
	]
)

main = runtime
`))
	require.NoError(t, err)

	scriptB, err := Unmarshal([]byte(
		checkoutInfo(testProject, mergeBTime) + `
runtime = solve(
	at_time = at_time,
	platforms = [
		"12345",
		"67890"
	],
	requirements = [
		Req(name = "perl", namespace = "language"),
		Req(name = "JSON", namespace = "language/perl")
	]
)

main = runtime
`))
	require.NoError(t, err)

	strategies := &mono_models.MergeStrategies{
		OverwriteChanges: []*mono_models.CommitChangeEditable{
			{Namespace: "language/perl", Requirement: "JSON", Operation: mono_models.CommitChangeEditableOperationAdded},
		},
	}

	require.True(t, isAutoMergePossible(scriptA, scriptB))

	err = scriptA.Merge(scriptB, strategies)
	require.NoError(t, err)

	v, err := scriptA.Marshal()
	require.NoError(t, err)

	assert.Equal(t,
		checkoutInfo(testProject, mergeBTime)+`
runtime = solve(
	at_time = at_time,
	platforms = [
		"12345",
		"67890"
	],
	requirements = [
		Req(name = "perl", namespace = "language"),
		Req(name = "DateTime", namespace = "language/perl"),
		Req(name = "JSON", namespace = "language/perl")
	]
)

main = runtime`, string(v))
}

func TestMergeRemove(t *testing.T) {
	scriptA, err := Unmarshal([]byte(
		checkoutInfo(testProject, mergeBTime) + `
runtime = solve(
	at_time = at_time,
	platforms = [
		"12345",
		"67890"
	],
	requirements = [
		Req(name = "perl", namespace = "language"),
		Req(name = "JSON", namespace = "language/perl"),
		Req(name = "DateTime", namespace = "language/perl")
	]
)

main = runtime
`))
	require.NoError(t, err)

	scriptB, err := Unmarshal([]byte(
		checkoutInfo(testProject, mergeATime) + `
runtime = solve(
	at_time = at_time,
	platforms = [
		"12345",
		"67890"
	],
	requirements = [
		Req(name = "perl", namespace = "language"),
		Req(name = "DateTime", namespace = "language/perl")
	]
)

main = runtime
`))

	strategies := &mono_models.MergeStrategies{
		OverwriteChanges: []*mono_models.CommitChangeEditable{
			{Namespace: "language/perl", Requirement: "JSON", Operation: mono_models.CommitChangeEditableOperationRemoved},
		},
	}

	require.True(t, isAutoMergePossible(scriptA, scriptB))

	err = scriptA.Merge(scriptB, strategies)
	require.NoError(t, err)

	v, err := scriptA.Marshal()
	require.NoError(t, err)

	assert.Equal(t,
		checkoutInfo(testProject, mergeBTime)+`
runtime = solve(
	at_time = at_time,
	platforms = [
		"12345",
		"67890"
	],
	requirements = [
		Req(name = "perl", namespace = "language"),
		Req(name = "DateTime", namespace = "language/perl")
	]
)

main = runtime`, string(v))
}

func TestMergeConflict(t *testing.T) {
	scriptA, err := Unmarshal([]byte(
		checkoutInfo(testProject, mergeATime) + `
runtime = solve(
	at_time = at_time,
	platforms = [
		"12345",
		"67890"
	],
	requirements = [
		Req(name = "perl", namespace = "language")
	]
)

main = runtime
`))
	require.NoError(t, err)

	scriptB, err := Unmarshal([]byte(
		checkoutInfo(testProject, mergeATime) + `
runtime = solve(
	at_time = at_time,
	platforms = [
		"12345"
	],
	requirements = [
		Req(name = "perl", namespace = "language"),
		Req(name = "JSON", namespace = "language/perl")
	]
)

main = runtime
`))
	require.NoError(t, err)

	assert.False(t, isAutoMergePossible(scriptA, scriptB)) // platforms do not match

	err = scriptA.Merge(scriptB, nil)
	require.Error(t, err)
}

func TestDeleteKey(t *testing.T) {
	m := map[string]interface{}{"foo": map[string]interface{}{"bar": "baz", "quux": "foobar"}}
	assert.True(t, deleteKey(&m, "quux"), "did not find quux")
	_, exists := m["foo"].(map[string]interface{})["quux"]
	assert.False(t, exists, "did not delete quux")
}
