package installer_test

import (
	"bytes"
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/ActiveState/cli/cmd/state-installer/internal/installer"
	"github.com/phayes/permbits"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var stateToolTestFile string = "state"
var otherTestFile string = "other"
var installerTestFile string = "state-installer"

var stateToolTestFileContent []byte = []byte("#!/bin/bash\necho I want to be replaced;")
var updatedTestFileContent []byte = []byte("#!/bin/bash\necho updated;")

func init() {
	if runtime.GOOS == "windows" {
		stateToolTestFile = "state.exe"
		otherTestFile = "other.exe"
		installerTestFile = "state-installer.exe"
	}
}

func copyStateToolTestFile(t *testing.T, targetPath string) {
	err := ioutil.WriteFile(targetPath, stateToolTestFileContent, 0755)
	require.NoError(t, err)
}

func initTempInstallDirs(t *testing.T) (string, string) {
	fromDir, err := ioutil.TempDir("", "from*")
	require.NoError(t, err)
	toDir, err := ioutil.TempDir("", "to*")
	require.NoError(t, err)
	for _, df := range []string{otherTestFile, stateToolTestFile} {
		// populate fromDir with a file that is going to be installed
		err = ioutil.WriteFile(filepath.Join(fromDir, df), updatedTestFileContent, 0775)
		require.NoError(t, err, "Failed to write test file %s", df)

	}
	// populate State Tool test file that gets replaced in installation directory
	copyStateToolTestFile(t, filepath.Join(toDir, stateToolTestFile))

	return fromDir, toDir
}

func assertPermissions(t *testing.T, fp string) {
	info, err := os.Stat(fp)
	require.NoError(t, err)
	pb := permbits.FileMode(info.Mode())
	assert.True(t, pb.UserRead(), "%s should be readable")
	if runtime.GOOS != "windows" {
		// Windows does not need an executable flag (just the correct file ending)
		assert.True(t, pb.UserExecute(), "%s should be executable")
	}
}

func assertSuccessfulInstallation(t *testing.T, toDir string) {
	for _, df := range []string{stateToolTestFile, otherTestFile} {
		fp := filepath.Join(toDir, df)
		assert.FileExists(t, fp, "Expected test file %s to exist", fp)
		b, err := ioutil.ReadFile(fp)
		require.NoError(t, err)
		if !bytes.Equal(updatedTestFileContent, b) {
			t.Errorf("Test file %s was not correctly updated", fp)
		}
		assertPermissions(t, fp)
	}
}

func assertRevertedInstallation(t *testing.T, toDir string) {
	fp := filepath.Join(toDir, stateToolTestFile)
	b, err := ioutil.ReadFile(fp)
	require.NoError(t, err)
	if !bytes.Equal(stateToolTestFileContent, b) {
		t.Error("State Tool test file was not correctly restored.")
	}
	assertPermissions(t, fp)
}

// TestInstallation tests that an installation is working if there are no obstacles like running processes
func TestInstallation(t *testing.T) {
	tests := []struct {
		Name                      string
		SimulateAdminInstallation bool
		ExpectSuccess             bool
	}{
		{"successful", false, true},
		{
			"update-without-permissions",
			true,
			// On Windows, the installation will succeed even if the existing files have no write permissions, because we are allowed to rename them and delete them (we are just not allowed to overwrite them!).
			runtime.GOOS == "windows",
		},
	}

	for _, tt := range tests {
		t.Run(tt.Name, func(t *testing.T) {
			from, to := initTempInstallDirs(t)

			if tt.SimulateAdminInstallation {
				// Simulate that a previous installation has been installed with administrator rights:
				// Remove the "Writable"-permission for installed files
				err := os.Chmod(to, 0550)
				require.NoError(t, err)
				err = os.Chmod(filepath.Join(to, stateToolTestFile), 0550)
				require.NoError(t, err)
			}

			inst := installer.New(from, to)
			err := inst.Install()

			if tt.ExpectSuccess {
				require.NoError(t, err)

				err = inst.RemoveBackupFiles()
				require.NoError(t, err)

				assertSuccessfulInstallation(t, to)
			} else {
				require.Error(t, err)

				err = inst.RemoveBackupFiles()
				require.NoError(t, err)

				assertRevertedInstallation(t, to)
			}

		})

	}
}
