package new

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/internal/testhelpers/exiter"
	"github.com/ActiveState/cli/internal/testhelpers/httpmock"
	"github.com/ActiveState/cli/pkg/platform/api"
	"github.com/ActiveState/cli/pkg/platform/authentication"
)

func init() {
	exit = exiter.Exit
}

// Runs "state new test-name -o test-owner -v 1.0" in an empty directory.
// Verifies that a project was successfully created.
func TestNewInEmptyDir(t *testing.T) {
	tmpdir, err := ioutil.TempDir("", "cli-new-test")
	assert.NoError(t, err, "Created temp directory")
	cwd, err := os.Getwd()
	assert.NoError(t, err, "Fetched cwd")

	httpmock.Activate(api.GetServiceURL(api.ServiceMono).String())
	defer httpmock.DeActivate()

	httpmock.Register("POST", "/login")
	authentication.Get().AuthenticateWithToken("")

	httpmock.Register("GET", "/organizations")
	httpmock.Register("POST", "organizations/test-owner/projects")

	err = os.Chdir(tmpdir)
	assert.NoError(t, err, "Switched to tempdir")

	Cc := Command.GetCobraCmd()
	Cc.SetArgs([]string{"test-name", "-o", "test-owner", "-v", "1.0"})
	err = Command.Execute()

	assert.NoError(t, err, "Executed without error")
	assert.NoError(t, failures.Handled(), "No failure occurred")

	_, err = os.Stat(filepath.Join(tmpdir, constants.ConfigFileName))
	assert.NoError(t, err, "Project was created")
	err = os.Rename(constants.ConfigFileName, constants.ConfigFileName+".bak")
	assert.NoError(t, err, "Renamed config file so later tests cannot reference it")
	os.Chdir(cwd) // restore

	os.RemoveAll(tmpdir)
}

// Runs "state new test-name -o test-owner -v 1.0" in a non-empty directory of
// just files.
// Verifies that a project was successfully created in a sub-directory.
func TestNewInNonEmptyDir(t *testing.T) {
	Flags.Path, Flags.Owner, Flags.Version, Args.Name = "", "", "", "" // reset
	tmpdir, _ := ioutil.TempDir("", "cli-new-test")
	cwd, _ := os.Getwd()
	err := ioutil.WriteFile(filepath.Join(tmpdir, "foo.txt"), []byte(""), 0666)
	assert.NoError(t, err, "Wrote dummy file")

	httpmock.Activate(api.GetServiceURL(api.ServiceMono).String())
	defer httpmock.DeActivate()

	httpmock.Register("POST", "/login")
	authentication.Get().AuthenticateWithToken("")

	httpmock.Register("GET", "/organizations")
	httpmock.Register("POST", "organizations/test-owner/projects")

	os.Chdir(tmpdir)
	Cc := Command.GetCobraCmd()
	Cc.SetArgs([]string{"test-name", "-o", "test-owner", "-v", "1.0"})
	err = Command.Execute()

	assert.NoError(t, err, "Executed without error")
	assert.NoError(t, failures.Handled(), "No failure occurred")

	_, err = os.Stat(filepath.Join(tmpdir, "test-name", constants.ConfigFileName))
	assert.NoError(t, err, "Project was created in sub-directory")
	os.Chdir(cwd) // restore

	os.RemoveAll(tmpdir)
}

// Runs "state new test-name -o test-owner -v 1.0" in a non-empty directory of
// files and folders.
// Verifies that a project was NOT created in a sub-directory due to a name
// conflict.
func TestNewInNonEmptyDirFail(t *testing.T) {
	Flags.Path, Flags.Owner, Flags.Version, Args.Name = "", "", "", "" // reset
	tmpdir, _ := ioutil.TempDir("", "cli-new-test")
	cwd, _ := os.Getwd()
	err := ioutil.WriteFile(filepath.Join(tmpdir, "foo.txt"), []byte(""), 0666)
	assert.NoError(t, err, "Wrote dummy file")
	err = os.Mkdir(filepath.Join(tmpdir, "test-name"), 0755)
	assert.NoError(t, err, "Wrote dummy directory")

	httpmock.Activate(api.GetServiceURL(api.ServiceMono).String())
	defer httpmock.DeActivate()

	httpmock.Register("POST", "/login")
	authentication.Get().AuthenticateWithToken("")

	httpmock.Register("GET", "/organizations")
	httpmock.Register("POST", "organizations/test-owner/projects")

	os.Chdir(tmpdir)
	Cc := Command.GetCobraCmd()
	Cc.SetArgs([]string{"test-name", "-o", "test-owner", "-v", "1.0"})

	code := exiter.WaitForExit(func() {
		Command.Execute()
	})
	assert.Equal(t, 1, code, "Exited with code 1")

	_, err = os.Stat(filepath.Join(tmpdir, "test-name", constants.ConfigFileName))
	assert.Error(t, err, "Project was not created in existing sub-directory")
	os.Chdir(cwd) // restore

	os.RemoveAll(tmpdir)
}

// Runs "state new test-name -p tmpdir -o test-owner -v 1.0".
// Verifies that a project was successfully created in tmpdir.
func TestNewWithPathToExistingDir(t *testing.T) {
	Flags.Path, Flags.Owner, Flags.Version, Args.Name = "", "", "", "" // reset
	tmpdir, _ := ioutil.TempDir("", "cli-new-test")
	cwd, _ := os.Getwd()
	err := ioutil.WriteFile(filepath.Join(tmpdir, "foo.txt"), []byte(""), 0666)
	assert.NoError(t, err, "Wrote dummy file")

	httpmock.Activate(api.GetServiceURL(api.ServiceMono).String())
	defer httpmock.DeActivate()

	httpmock.Register("POST", "/login")
	authentication.Get().AuthenticateWithToken("")

	httpmock.Register("GET", "/organizations")
	httpmock.Register("POST", "organizations/test-owner/projects")

	os.Chdir(tmpdir)
	Cc := Command.GetCobraCmd()
	Cc.SetArgs([]string{"test-name", "-p", tmpdir, "-o", "test-owner", "-v", "1.0"})

	code := exiter.WaitForExit(func() {
		Command.Execute()
	})
	assert.Equal(t, 1, code, "Exited with code 1")

	_, err = os.Stat(filepath.Join(tmpdir, constants.ConfigFileName))
	assert.Error(t, err, "Project was not created in existing directory")
	os.Chdir(cwd) // restore

	os.RemoveAll(tmpdir)
}

// Runs "state new test-name -o test-owner -v badVersion".
// Verifies that a project was NOT created due to a bad version number.
func TestNewWithBadVersion(t *testing.T) {
	Flags.Path, Flags.Owner, Flags.Version, Args.Name = "", "", "", "" // reset
	tmpdir, _ := ioutil.TempDir("", "cli-new-test")
	cwd, _ := os.Getwd()

	os.Chdir(tmpdir)
	Cc := Command.GetCobraCmd()
	Cc.SetArgs([]string{"test-name", "-o", "test-owner", "-v", "badVersion"})

	code := exiter.WaitForExit(func() {
		Command.Execute()
	})
	assert.Equal(t, 1, code, "Exited with code 1")

	_, err := os.Stat(filepath.Join(tmpdir, constants.ConfigFileName))
	assert.Error(t, err, "Project was not created")
	os.Chdir(cwd) // restore

	os.RemoveAll(tmpdir)
}

// Runs "state new test-name -v 1.0" in an empty directory.
// Verifies that a project was successfully created with an owner fetched from
// the Platform.
func TestNewWithNoOwner(t *testing.T) {
	Flags.Path, Flags.Owner, Flags.Version, Args.Name = "", "", "", "" // reset
	tmpdir, _ := ioutil.TempDir("", "cli-new-test")
	cwd, _ := os.Getwd()

	httpmock.Activate(api.GetServiceURL(api.ServiceMono).String())
	defer httpmock.DeActivate()

	httpmock.Register("POST", "/login")
	authentication.Get().AuthenticateWithToken("")

	httpmock.Register("GET", "/organizations")
	httpmock.Register("POST", "organizations/test-owner/projects")

	os.Chdir(tmpdir)
	Cc := Command.GetCobraCmd()
	Cc.SetArgs([]string{"test-name", "-v", "1.0"})
	err := Command.Execute()

	assert.NoError(t, err, "Executed without error")
	assert.NoError(t, failures.Handled(), "No failure occurred")

	_, err = os.Stat(filepath.Join(tmpdir, constants.ConfigFileName))
	assert.NoError(t, err, "Project was created")
	os.Chdir(cwd) // restore

	os.RemoveAll(tmpdir)
}

// Runs "state new test-name -v 1.0" in an empty directory, but test-name
// happens to already exist on the Platform.
// Verifies that a project was NOT created due to a name conflict.
func TestNewPlatformProjectExists(t *testing.T) {
	Flags.Path, Flags.Owner, Flags.Version, Args.Name = "", "", "", "" // reset
	tmpdir, _ := ioutil.TempDir("", "cli-new-test")
	cwd, _ := os.Getwd()

	httpmock.Activate(api.GetServiceURL(api.ServiceMono).String())
	defer httpmock.DeActivate()

	httpmock.Register("POST", "/login")
	authentication.Get().AuthenticateWithToken("")

	httpmock.Register("GET", "/organizations")
	httpmock.Register("GET", "/organizations/test-owner/projects/test-name")

	os.Chdir(tmpdir)
	Cc := Command.GetCobraCmd()
	Cc.SetArgs([]string{"test-name", "-v", "1.0"})

	code := exiter.WaitForExit(func() {
		Command.Execute()
	})
	assert.Equal(t, 1, code, "Exited with code 1")

	_, err := os.Stat(filepath.Join(tmpdir, constants.ConfigFileName))
	assert.Error(t, err, "Platform project exists; project was not created")
	os.Chdir(cwd) // restore

	os.RemoveAll(tmpdir)
}
