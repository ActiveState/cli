package buildscript

import (
	"testing"

	"github.com/ActiveState/cli/internal/rtutils/ptr"
	"github.com/ActiveState/cli/pkg/platform/runtime/buildscript"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDiff(t *testing.T) {
	script, err := buildscript.NewScript([]byte(
		`let:
	runtime = solve(
		platforms = [
			"12345",
			"67890"
		],
		requirements = [
			{
				name = "language/python",
				namespace = "language"
			}
		]
	)

in:
	runtime`))
	require.NoError(t, err)

	expr, err := script.ToBuildExpression()
	require.NoError(t, err)

	(*script.Let.Assignments[0].Value.FuncCall.Arguments[0].Assignment.Value.List)[0].Str = ptr.To(`"77777"`)

	result, err := generateDiff(script, expr)
	require.NoError(t, err)
	assert.Equal(t, `let:
	runtime = solve(
		platforms = [
<<<<<<< local
			"77777",
=======
			"12345",
>>>>>>> remote
			"67890"
		],
		requirements = [
			{
				name = "language/python",
				namespace = "language"
			}
		]
	)

in:
	runtime`, result)
}