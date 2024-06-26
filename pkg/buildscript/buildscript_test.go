package buildscript

import (
	"testing"
	"time"

	"github.com/go-openapi/strfmt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestRoundTripFromBuildScript tests that if we read a buildscript from disk and then write it again it produces the
// exact same value.
func TestRoundTripFromBuildScript(t *testing.T) {
	script, err := Unmarshal(basicBuildScript)
	require.NoError(t, err)

	data, err := script.Marshal()
	require.NoError(t, err)
	t.Logf("marshalled:\n%s\n---", string(data))

	roundTripScript, err := Unmarshal(data)
	require.NoError(t, err)

	assert.Equal(t, script, roundTripScript)
}

// TestRoundTripFromBuildExpression tests that if we receive a build expression from the API and eventually write it
// back without any modifications it is still the same.
func TestRoundTripFromBuildExpression(t *testing.T) {
	script, err := UnmarshalBuildExpression(basicBuildExpression, nil)
	require.NoError(t, err)
	data, err := script.MarshalBuildExpression()
	require.NoError(t, err)
	t.Logf("marshalled:\n%s\n---", string(data))
	require.Equal(t, string(basicBuildExpression), string(data))
}

func TestExpressionToScript(t *testing.T) {
	ts, err := time.Parse(strfmt.RFC3339Millis, atTime)
	require.NoError(t, err)

	script, err := UnmarshalBuildExpression(basicBuildExpression, &ts)
	require.NoError(t, err)

	data, err := script.Marshal()
	require.NoError(t, err)
	t.Logf("marshalled:\n%s\n---", string(data))
	require.Equal(t, string(basicBuildScript), string(data))
}

func TestScriptToExpression(t *testing.T) {
	bs, err := Unmarshal(basicBuildScript)
	require.NoError(t, err)

	as, err := bs.MarshalBuildExpression()
	require.NoError(t, err)

	require.Equal(t, string(basicBuildExpression), string(as))
}
