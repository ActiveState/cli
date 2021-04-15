package installer_test

import (
	"bytes"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/ActiveState/cli/cmd/state-installer/internal/installer"
	"github.com/ActiveState/cli/internal/config"
	"github.com/ActiveState/cli/internal/environment"
	"github.com/ActiveState/cli/internal/exeutils"
	"github.com/ActiveState/cli/internal/installation"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var stateToolTestFile string = "state"
var otherTestFile string = "other"
var installerTestFile string = "installer"

var stateToolTestFileContent []byte
var installerTestFileContent []byte
var updatedTestFileContent []byte = []byte("#!/bin/bash\necho updated;")

func init() {
	if runtime.GOOS == "windows" {
		stateToolTestFile = "state.exe"
		otherTestFile = "other.exe"
		installerTestFile = "installer.exe"
	}
}

func buildTestExecutable(t *testing.T, dir, exe string) {
	root, err := environment.GetRootPath()
	require.NoError(t, err)

	cmd := exec.Command(
		"go", "build", "-o", exe,
		filepath.Join(root, "cmd", "state-installer", "internal", "installer", dir),
	)
	err = cmd.Run()
	require.NoError(t, err)
}

func copyStateToolTestFile(t *testing.T, targetPath string) {
	if stateToolTestFileContent == nil {
		td, err := ioutil.TempDir("", "")
		require.NoError(t, err)
		fp := filepath.Join(td, stateToolTestFile)

		buildTestExecutable(t, "testcmd", fp)
		stateToolTestFileContent, err = ioutil.ReadFile(fp)
		require.NoError(t, err)
	}
	err := ioutil.WriteFile(targetPath, stateToolTestFileContent, 0755)
	require.NoError(t, err)
}

func copyInstallerTestFile(t *testing.T, targetPath string) {
	if installerTestFileContent == nil {
		td, err := ioutil.TempDir("", "")
		require.NoError(t, err)
		fp := filepath.Join(td, installerTestFile)

		buildTestExecutable(t, "testinst", fp)
		installerTestFileContent, err = ioutil.ReadFile(fp)
		require.NoError(t, err)
	}
	err := ioutil.WriteFile(targetPath, installerTestFileContent, 0755)
	require.NoError(t, err)
}

func initTempInstallDirs(t *testing.T, withAutoInstall bool) (string, string) {
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

	if withAutoInstall {
		copyInstallerTestFile(t, filepath.Join(fromDir, installerTestFile))
	}

	return fromDir, toDir
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
		info, err := os.Stat(fp)
		require.NoError(t, err)
		assert.Equal(t, "rwx", info.Mode().String()[1:4])
	}
}

func assertRevertedInstallation(t *testing.T, toDir string) {
	fp := filepath.Join(toDir, stateToolTestFile)
	b, err := ioutil.ReadFile(fp)
	require.NoError(t, err)
	if !bytes.Equal(stateToolTestFileContent, b) {
		t.Error("State Tool test file was not correctly restored.")
	}
}

// TestInstallation tests that an installation is working if there are no obstacles like running processes
func TestInstallation(t *testing.T) {
	tests := []struct {
		Name                      string
		SimulateAdminInstallation bool
		ExpectSuccess             bool
	}{
		{"successful", false, true},
		{"update-without-permissions", true, false},
	}

	for _, tt := range tests {
		t.Run(tt.Name, func(t *testing.T) {
			from, to := initTempInstallDirs(t, false)

			if tt.SimulateAdminInstallation {
				// Simulate that a previous installation has been installed with administrator rights:
				// Remove the "Writable"-permission for installed files
				err := os.Chmod(to, 0550)
				require.NoError(t, err)
				err = os.Chmod(filepath.Join(to, stateToolTestFile), 0550)
				require.NoError(t, err)
			}

			err := installer.Install(from, to)

			if tt.ExpectSuccess {
				require.NoError(t, err)
				assertSuccessfulInstallation(t, to)
			} else {
				assert.Error(t, err)
				assertRevertedInstallation(t, to)
			}
		})

	}
}

func TestInstallationWhileProcessesAreActive(t *testing.T) {
	from, to := initTempInstallDirs(t, false)

	// run the old command which waits for one second.
	cmd := exec.Command(filepath.Join(to, stateToolTestFile), "1")
	err := cmd.Start()
	require.NoError(t, err)

	errC := make(chan error)
	go func() {
		errC <- installer.Install(from, to)
	}()

	err = cmd.Wait()
	require.NoError(t, err)

	select {
	case err := <-errC:
		if runtime.GOOS == "windows" {
			assert.Error(t, err, "Installation should fail on Windows.")
			assertRevertedInstallation(t, to)
		} else {
			require.NoError(t, err)
			assertSuccessfulInstallation(t, to)
		}
	case <-time.After(time.Second * 2):
		t.Fatalf("Timeout waiting for installation to finish")
	}

}

// TestAutoUpdate tests that an executable can update itself, by spawning the installer process which eventually replaces the calling executable.
func TestAutoUpdate(t *testing.T) {
	tests := []struct {
		Name          string
		Timeout       string
		ExpectSuccess bool
	}{
		{
			"replaced-executable-is-running",
			"0",
			// when the replaced executable is still running, the auto-update should fail on Windows
			runtime.GOOS != "windows",
		},
		{
			"replaced-executable-shut-down",
			"2",
			// when the replaced executable is stopped, the auto-update should always pass
			true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.Name, func(t *testing.T) {
			from, to := initTempInstallDirs(t, true)
			defer os.RemoveAll(from)
			defer os.RemoveAll(to)

			// run installer
			stdout, stderr, err := exeutils.ExecSimple(
				filepath.Join(to, stateToolTestFile), from, filepath.Join(from, installerTestFile), tt.Timeout)
			require.NoError(t, err, "Error running auto-replacing test file: %v, stderr=%s", err, stderr)

			pid, err := strconv.ParseInt(stdout, 10, 32)
			require.NoError(t, err)
			assert.NotEqual(t, 0, pid)

			cfg, err := config.Get()
			require.NoError(t, err)
			logFile := installation.LogfilePath(cfg.ConfigPath(), int(pid))

			// poll for successful auto-update
			for i := 0; i < 20; i++ {
				time.Sleep(time.Millisecond * 200)

				logs, err := ioutil.ReadFile(logFile)
				require.NoError(t, err)
				if strings.Contains(string(logs), "was successful") || strings.Contains(string(logs), "Installation failed") {
					break
				}
			}

			logs, err := ioutil.ReadFile(logFile)
			require.NoError(t, err)

			if tt.ExpectSuccess {
				assert.Containsf(t, string(logs), "was successful", "logs should contain 'was successful', got=%s", string(logs))
				assert.Contains(t, string(logs), "Target files=", "logs should contain 'Target files=', got=%s", logs)

				assertSuccessfulInstallation(t, to)
			} else {
				assert.Containsf(t, string(logs), "Installation failed", "logs should contains 'Installation failed', got=%s", string(logs))
				assert.Contains(t, string(logs), "Successfully restored original files.")

				assertRevertedInstallation(t, to)
			}
		})
	}
}
