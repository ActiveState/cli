package buildscript

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/pkg/platform/runtime/buildscript"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDiff(t *testing.T) {
	script, err := buildscript.New([]byte(
		`at_time = "2000-01-01T00:00:00.000Z"
runtime = solve(
	at_time = at_time,
	platforms = [
		"12345",
		"67890"
	],
	requirements = [
		Req(name = "language/perl")
	]
)

main = runtime`))
	require.NoError(t, err)

	// Modify the build script.
	modifiedScript, err := buildscript.New([]byte(strings.Replace(script.String(), "12345", "77777", 1)))
	require.NoError(t, err)

	// Generate the difference between the modified script and the original expression.
	result, err := generateDiff(modifiedScript, script)
	require.NoError(t, err)
	assert.Equal(t, `at_time = "2000-01-01T00:00:00.000Z"
runtime = solve(
	at_time = at_time,
	platforms = [
<<<<<<< local
		"77777",
=======
		"12345",
>>>>>>> remote
		"67890"
	],
	requirements = [
		Req(name = "language/perl")
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
	script1, err := buildscript.New(fileutils.ReadFileUnsafe(filepath.Join("testdata", "buildscript1.as")))
	require.NoError(t, err)
	script2, err := buildscript.New(fileutils.ReadFileUnsafe(filepath.Join("testdata", "buildscript2.as")))
	require.NoError(t, err)
	result, err := generateDiff(script1, script2)
	require.NoError(t, err)
	assert.Equal(t, `<<<<<<< local
at_time = "2023-10-16T22:20:29.000Z"
=======
at_time = "2023-08-01T16:20:11.985Z"
>>>>>>> remote
runtime = state_tool_artifacts_v1(
	build_flags = [
	],
	camel_flags = [
	],
	src = sources
)
sources = solve(
	at_time = at_time,
	platforms = [
		"78977bc8-0f32-519d-80f3-9043f059398c",
		"7c998ec2-7491-4e75-be4d-8885800ef5f2",
		"96b7e6f2-bebf-564c-bc1c-f04482398f38"
	],
	requirements = [
		Req(name = "language/python", version = Eq(value = "3.10.11")),
<<<<<<< local
		Req(name = "language/python/requests")
=======
		Req(name = "language/python/requests", version = Eq(value = "2.30.0"))
>>>>>>> remote
	],
	solver_version = null
)

main = runtime`, result)
}
