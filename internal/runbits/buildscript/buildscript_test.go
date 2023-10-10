package buildscript

import (
	"encoding/json"
	"testing"

	"github.com/ActiveState/cli/internal/rtutils/ptr"
	"github.com/ActiveState/cli/pkg/platform/runtime/buildexpression"
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

	// Make a copy of the original expression.
	bytes, err := json.Marshal(script.Expr)
	require.NoError(t, err)
	expr, err := buildexpression.New(bytes)
	require.NoError(t, err)

	// Modify the build script.
	(*script.Expr.Let.Assignments[0].Value.Ap.Arguments[0].Assignment.Value.List)[0].Str = ptr.To(`77777`)

	// Generate the difference between the modified script and the original expression.
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
