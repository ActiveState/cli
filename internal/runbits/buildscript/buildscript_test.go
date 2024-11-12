package buildscript_runbit

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/pkg/buildscript"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testProject = "https://platform.activestate.com/org/project?branch=main&commitID=00000000-0000-0000-0000-000000000000"
const testTime = "2000-01-01T00:00:00Z"

func checkoutInfo(project, time string) string {
	return "```\n" +
		"Project: " + project + "\n" +
		"Time: " + time + "\n" +
		"```\n"
}

func TestDiff(t *testing.T) {
	script, err := buildscript.Unmarshal([]byte(
		checkoutInfo(testProject, testTime) + `
runtime = solve(
	at_time = TIME,
	platforms = [
		"12345",
		"67890"
	],
	requirements = [
		Req(name = "perl", namespace = "language")
	]
)

main = runtime`))
	require.NoError(t, err)

	bs, err := script.Marshal()
	require.NoError(t, err)

	// Modify the build script.
	modifiedScript, err := buildscript.Unmarshal([]byte(strings.Replace(string(bs), "12345", "77777", 1)))
	require.NoError(t, err)

	// Generate the difference between the modified script and the original expression.
	result, err := generateDiff(modifiedScript, script)
	require.NoError(t, err)
	assert.Equal(t, checkoutInfo(testProject, testTime)+`
runtime = solve(
	at_time = TIME,
	platforms = [
<<<<<<< local
		"77777",
=======
		"12345",
>>>>>>> remote
		"67890"
	],
	requirements = [
		Req(name = "perl", namespace = "language")
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
	script1, err := buildscript.Unmarshal(fileutils.ReadFileUnsafe(filepath.Join("testdata", "buildscript1.as")))
	require.NoError(t, err)
	script2, err := buildscript.Unmarshal(fileutils.ReadFileUnsafe(filepath.Join("testdata", "buildscript2.as")))
	require.NoError(t, err)
	result, err := generateDiff(script1, script2)
	require.NoError(t, err)
	assert.Equal(t,
		"```\n"+
			"<<<<<<< local\n"+
			"Project: https://platform.activestate.com/ActiveState-CLI/Merge?branch=main&commitID=d908a758-6a81-40d4-b0eb-87069cd7f07d\n"+
			"Time: 2024-05-10T00:00:13Z\n"+
			"=======\n"+
			"Project: https://platform.activestate.com/ActiveState-CLI/Merge?branch=main&commitID=f3263ee4-ac4c-41ee-b778-2585333f49f7\n"+
			"Time: 2023-08-01T16:20:11Z\n"+
			">>>>>>> remote\n"+
			"```\n"+`
runtime = state_tool_artifacts_v1(
	build_flags = [
	],
	camel_flags = [
	],
	src = sources
)
sources = solve(
	at_time = TIME,
	platforms = [
		"78977bc8-0f32-519d-80f3-9043f059398c",
		"7c998ec2-7491-4e75-be4d-8885800ef5f2",
		"96b7e6f2-bebf-564c-bc1c-f04482398f38"
	],
	requirements = [
		Req(name = "python", namespace = "language", version = Eq(value = "3.10.11")),
<<<<<<< local
		Req(name = "requests", namespace = "language/python")
=======
		Req(name = "requests", namespace = "language/python", version = Eq(value = "2.30.0"))
>>>>>>> remote
	],
	solver_version = null
)

main = runtime`, result)
}
