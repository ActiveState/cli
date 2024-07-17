package buildscript

import (
	"fmt"
	"testing"
	"time"

	"github.com/go-openapi/strfmt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var atTime = "2000-01-01T00:00:00.000Z"

var basicBuildScript = []byte(fmt.Sprintf(
	`at_time = "%s"
runtime = state_tool_artifacts(
	src = sources
)
sources = solve(
	at_time = at_time,
	platforms = [
		"12345",
		"67890"
	],
	requirements = [
		Req(name = "python", namespace = "language", version = Eq(value = "3.10.10"))
	],
	solver_version = null
)

main = runtime`, atTime))

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
	script, err := UnmarshalBuildExpression(basicBuildExpression, nil)
	require.NoError(t, err)

	data, err := script.MarshalBuildExpression()
	require.NoError(t, err)

	require.Equal(t, string(basicBuildExpression), string(data))
}

// TestExpressionToScript tests that creating a build script from a given Platform build expression
// and at time produces the expected result.
func TestExpressionToScript(t *testing.T) {
	ts, err := time.Parse(strfmt.RFC3339Millis, atTime)
	require.NoError(t, err)

	script, err := UnmarshalBuildExpression(basicBuildExpression, &ts)
	require.NoError(t, err)

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
