package buildscript

import (
	"encoding/json"
	"path/filepath"
	"testing"

	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/rtutils/ptr"
	"github.com/ActiveState/cli/pkg/platform/runtime/buildexpression"
	"github.com/ActiveState/cli/pkg/platform/runtime/buildscript"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDiff(t *testing.T) {
	script, err := buildscript.NewScript([]byte(
		`runtime = solve(
	platforms = [
		"12345",
		"67890"
	],
	requirements = [
		Req(name="language/perl")
	]
)

main = runtime`))
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
	assert.Equal(t, `runtime = solve(
	platforms = [
<<<<<<< local
		"77777",
=======
		"12345",
>>>>>>> remote
		"67890"
	],
	requirements = [
		Req(name="language/perl")
	]
)

main = runtime`, result)
}

// TestRealWorld tests a real-world case where:
//   - There is a Platform Python project with an initial commit.
//   - There is a local project that just checks it out.
//   - The Platform project adds requests@2.30.0 (an older version).
//   - The local project adds requests (latest version).
//   - The local project pulls from the Platform project, resulting in conflicting times and version
//     requirements for requests.
func TestRealWorld(t *testing.T) {
	script1, err := buildscript.NewScript(fileutils.ReadFileUnsafe(filepath.Join("testdata", "buildscript1.as")))
	require.NoError(t, err)
	script2, err := buildscript.NewScript(fileutils.ReadFileUnsafe(filepath.Join("testdata", "buildscript2.as")))
	require.NoError(t, err)
	result, err := generateDiff(script1, script2.Expr)
	require.NoError(t, err)
	assert.Equal(t, `runtime = state_tool_artifacts_v1(
	build_flags = [
	],
	camel_flags = [
	],
	src = "$sources"
)
sources = solve(
<<<<<<< local
	at_time = "2023-10-16T22:20:29.000000Z",
=======
	at_time = "2023-08-01T16:20:11.985000Z",
>>>>>>> remote
	platforms = [
		"78977bc8-0f32-519d-80f3-9043f059398c",
		"7c998ec2-7491-4e75-be4d-8885800ef5f2",
		"96b7e6f2-bebf-564c-bc1c-f04482398f38"
	],
	requirements = [
		Req(name="language/python", version="3.10.11"),
<<<<<<< local
		Req(name="language/python/requests")
=======
		Req(name="language/python/requests", version="2.30.0")
>>>>>>> remote
	],
	solver_version = null
)

main = runtime`, result)
}
