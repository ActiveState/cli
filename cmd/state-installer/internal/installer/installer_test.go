package installer_test

import (
	"bytes"
	"io/ioutil"
	"log"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/ActiveState/cli/cmd/state-installer/internal/installer"
	"github.com/ActiveState/cli/internal/environment"
	"github.com/ActiveState/cli/internal/exeutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var dummyFiles []string = []string{"state", "other"}
var dummyStateToolContent []byte
var dummyInstallerContent []byte
var dummyContent []byte = []byte("#!/bin/bash\necho updated;")

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

func copyDummyStateTool(t *testing.T, targetPath string) {
	if dummyStateToolContent == nil {
		td, err := ioutil.TempDir("", "")
		require.NoError(t, err)
		fp := filepath.Join(td, "state")

		buildTestExecutable(t, "testcmd", fp)
		dummyStateToolContent, err = ioutil.ReadFile(fp)
		require.NoError(t, err)
	}
	err := ioutil.WriteFile(targetPath, dummyStateToolContent, 0755)
	require.NoError(t, err)
}

func copyDummyInstaller(t *testing.T, targetPath string) {
	if dummyInstallerContent == nil {
		td, err := ioutil.TempDir("", "")
		require.NoError(t, err)
		fp := filepath.Join(td, "installer")

		buildTestExecutable(t, "testinst", fp)
		dummyInstallerContent, err = ioutil.ReadFile(fp)
		require.NoError(t, err)
	}
	err := ioutil.WriteFile(targetPath, dummyInstallerContent, 0755)
	require.NoError(t, err)
}

func initTempInstallDirs(t *testing.T, withAutoInstall bool) (string, string) {
	fromDir, err := ioutil.TempDir("", "from*")
	require.NoError(t, err)
	toDir, err := ioutil.TempDir("", "to*")
	require.NoError(t, err)
	// populate from dir with files that are going to be installed
	for _, df := range dummyFiles {
		err = ioutil.WriteFile(filepath.Join(fromDir, df), dummyContent, 0775)
		require.NoError(t, err, "Failed to write dummy file %s", df)
	}

	// populate dummy State Tool file that get replaced in installation directory
	copyDummyStateTool(t, filepath.Join(toDir, "state"))

	if withAutoInstall {
		copyDummyInstaller(t, filepath.Join(fromDir, "installer"))
	}

	return fromDir, toDir
}

func assertSuccessfulInstallation(t *testing.T, toDir, logs string) {
	assert.Contains(t, logs, "Target files=", "logs should contain 'Target files=', got=%s", logs)

	for _, df := range dummyFiles {
		fp := filepath.Join(toDir, df)
		assert.FileExists(t, fp, "Expected dummy file %s to exist", fp)
		b, err := ioutil.ReadFile(fp)
		require.NoError(t, err)
		assert.Equal(t, dummyContent, b, "Dummy file %s was not correctly updated", fp)
	}
}

func assertRevertedInstallation(t *testing.T, toDir, logs string) {
	assert.Contains(t, logs, "Successfully restored original files.")

	fp := filepath.Join(toDir, "state")
	b, err := ioutil.ReadFile(fp)
	require.NoError(t, err)
	assert.Equal(t, dummyStateToolContent, b, "Dummy State Tool file was not correctly restored")
}

// TestAutoUpdate tests that an executable can update itself, by spawning the installer process which eventually replaces the calling executable.
func TestAutoUpdate(t *testing.T) {
	from, to := initTempInstallDirs(t, true)

	logFile := filepath.Join(to, "install.log")

	// run installer
	_, stderr, err := exeutils.ExecSimple(filepath.Join(to, "state"), from, filepath.Join(from, "installer"), logFile)
	require.NoError(t, err, "Error running dummy State Tool: %v, stderr=%s", err, stderr)

	// poll for successful auto-update
	for i := 0; i < 20; i++ {
		time.Sleep(time.Millisecond * 200)

		logs, err := ioutil.ReadFile(logFile)
		require.NoError(t, err)
		if strings.Contains(string(logs), "was successful") {
			break
		}
	}

	logs, err := ioutil.ReadFile(logFile)
	require.NoError(t, err)
	assert.Containsf(t, string(logs), "was successful", "logs should contain 'was successful', got=%s", string(logs))

	assertSuccessfulInstallation(t, to, string(logs))
}

// TestInstallation tests that an installation is working if there are no obstacles like running processes
func TestInstallation(t *testing.T) {
	from, to := initTempInstallDirs(t, false)

	buf := bytes.NewBuffer(make([]byte, 0, 1000))
	logger := log.New(buf, "noop", 0)

	err := installer.Install(from, to, logger)
	require.NoError(t, err)

	assertSuccessfulInstallation(t, to, buf.String())
}

func TestInstallationWhileProcessesAreActive(t *testing.T) {
	from, to := initTempInstallDirs(t, false)

	// run the old command which waits for one second.
	cmd := exec.Command(filepath.Join(to, "state"), "2")
	err := cmd.Start()
	require.NoError(t, err)

	buf := bytes.NewBuffer(make([]byte, 0, 1000))
	logger := log.New(buf, "noop", 0)
	errC := make(chan error)
	go func() {
		errC <- installer.Install(from, to, logger)
	}()

	err = cmd.Wait()
	require.NoError(t, err)

	select {
	case err := <-errC:
		require.NoError(t, err)
		assertSuccessfulInstallation(t, to, buf.String())
	case <-time.After(time.Second * 2):
		t.Fatalf("Timeout waiting for installation to finish")
	}

}
