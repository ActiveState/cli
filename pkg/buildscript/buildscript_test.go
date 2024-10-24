package buildscript

import (
	"fmt"
	"path/filepath"
	"testing"
	"time"

	"github.com/ActiveState/cli/internal/environment"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/go-openapi/strfmt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var basicBuildScript = []byte(
	checkoutInfoString(testProject, testTime) + `
runtime = state_tool_artifacts(
	src = sources
)
sources = solve(
	at_time = TIME,
	platforms = [
		"12345",
		"67890"
	],
	requirements = [
		Req(name = "python", namespace = "language", version = Eq(value = "3.10.10"))
	],
	solver_version = null
)

main = runtime`)

var basicBuildExpression = []byte(`{
  "let": {
    "in": "$runtime",
    "runtime": {
      "state_tool_artifacts": {
        "src": "$sources"
      }
    },
    "sources": {
      "solve": {
        "at_time": "$at_time",
        "platforms": [
          "12345",
          "67890"
        ],
        "requirements": [
          {
            "name": "python",
            "namespace": "language",
            "version_requirements": [
              {
                "comparator": "eq",
                "version": "3.10.10"
              }
            ]
          }
        ],
        "solver_version": null
      }
    }
  }
}`)

// TestRoundTripFromBuildScript tests that if we read a build script from disk and then write it
// again it produces the exact same value.
func TestRoundTripFromBuildScript(t *testing.T) {
	script, err := Unmarshal(basicBuildScript)
	require.NoError(t, err)

	data, err := script.Marshal()
	require.NoError(t, err)
	t.Logf("marshalled:\n%s\n---", string(data))

	roundTripScript, err := Unmarshal(data)
	require.NoError(t, err)

	assert.Equal(t, script, roundTripScript)
	equal, err := script.Equals(roundTripScript)
	require.NoError(t, err)
	assert.True(t, equal)
}

// TestRoundTripFromBuildExpression tests that if we construct a buildscript from a Platform build
// expression and then immediately construct another build expression from that build script, the
// build expressions are identical.
func TestRoundTripFromBuildExpression(t *testing.T) {
	script := New()
	err := script.UnmarshalBuildExpression(basicBuildExpression)
	require.NoError(t, err)

	data, err := script.MarshalBuildExpression()
	require.NoError(t, err)

	require.Equal(t, string(basicBuildExpression), string(data))
}

func TestRoundTripFromBuildExpressionWithLegacyAtTime(t *testing.T) {
	wd, err := environment.GetRootPath()
	require.NoError(t, err)

	initialTimeStamp := "2024-10-15T16:37:06.260Z"
	updatedTimeStamp := "2024-10-15T16:37:06.261Z"

	data, err := fileutils.ReadFile(filepath.Join(wd, "pkg", "buildscript", "testdata", "buildexpression-roundtrip-legacy.json"))
	require.NoError(t, err)

	// The initial build expression does not use the new at_time format
	assert.NotContains(t, string(data), "$at_time")
	assert.Contains(t, string(data), initialTimeStamp)

	script := New()
	require.NoError(t, script.UnmarshalBuildExpression(data))

	// Ensure that legacy at_time is preserved in the buildscript.
	atTime := script.AtTime()
	require.NotNil(t, atTime)
	require.Equal(t, initialTimeStamp, atTime.Format(strfmt.RFC3339Millis))

	data, err = script.MarshalBuildExpression()
	require.NoError(t, err)

	// When the build expression is unmarshalled it should now use the new at_time format
	assert.Contains(t, string(data), "$at_time")
	assert.NotContains(t, string(data), initialTimeStamp)

	// Update the time in the build script
	updatedTime, err := time.Parse(strfmt.RFC3339Millis, updatedTimeStamp)
	require.NoError(t, err)
	script.SetAtTime(updatedTime)

	// The updated time should be reflected in the build script
	require.Equal(t, updatedTime, *script.AtTime())

	data, err = script.Marshal()
	require.NoError(t, err)

	// The marshalled build script should now contain the updated time
	// in the Time block at the top of the script.
	assert.Contains(t, string(data), updatedTimeStamp)
	assert.NotContains(t, string(data), fmt.Sprintf("Time: %s", initialTimeStamp))

	data, err = script.MarshalBuildExpression()
	require.NoError(t, err)

	// The build expression representation should now use the new at_time format
	assert.Contains(t, string(data), "$at_time")
}

// TestExpressionToScript tests that creating a build script from a given Platform build expression
// and at time produces the expected result.
func TestExpressionToScript(t *testing.T) {
	ts, err := time.Parse(strfmt.RFC3339Millis, testTime)
	require.NoError(t, err)

	script := New()
	script.SetProject(testProject)
	script.SetAtTime(ts)
	require.NoError(t, script.UnmarshalBuildExpression(basicBuildExpression))

	data, err := script.Marshal()
	require.NoError(t, err)

	require.Equal(t, string(basicBuildScript), string(data))
}

// TestScriptToExpression tests that we can produce a valid Platform build expression from a build
// script on disk.
func TestScriptToExpression(t *testing.T) {
	bs, err := Unmarshal(basicBuildScript)
	require.NoError(t, err)

	data, err := bs.MarshalBuildExpression()
	require.NoError(t, err)

	require.Equal(t, string(basicBuildExpression), string(data))
}

func TestOutdatedScript(t *testing.T) {
	_, err := Unmarshal([]byte(
		`at_time = "2000-01-01T00:00:00.000Z"
	main = runtime
	`))
	assert.Error(t, err)
	assert.ErrorIs(t, err, ErrOutdatedAtTime)
}
