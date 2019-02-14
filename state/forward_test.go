package main

import (
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/pkg/projectfile"

	"github.com/ActiveState/cli/internal/environment"
	"github.com/ActiveState/cli/internal/testhelpers/httpmock"
	"github.com/ActiveState/cli/internal/testhelpers/osutil"
	"github.com/stretchr/testify/assert"
)

type ClosingBuffer struct {
	*bytes.Buffer
}

func (cb *ClosingBuffer) Close() (err error) {
	return
}

func testdataDir(t *testing.T) string {
	cwd, err := environment.GetRootPath()
	assert.NoError(t, err, "Should fetch cwd")
	return filepath.Join(cwd, "state", "testdata")
}

func setupCwd(t *testing.T, withVersion bool) {
	testdatadir := testdataDir(t)
	if withVersion {
		testdatadir = filepath.Join(testdatadir, "withversion")
	}
	err := os.Chdir(testdatadir)
	assert.NoError(t, err, "Should change dir without issue.")
	projectfile.Reset()
}

func mockUpdater(t *testing.T, version string) {
	var buf bytes.Buffer
	zw := gzip.NewWriter(&buf)
	zw.Name = constants.CommandName

	testdatadir := testdataDir(t)
	f, err := ioutil.ReadFile(filepath.Join(testdatadir, "state.sh"))
	assert.NoError(t, err)
	_, err = zw.Write(f)
	assert.NoError(t, err)
	assert.NoError(t, zw.Close())

	cb := &ClosingBuffer{bytes.NewBuffer(buf.Bytes())}
	h := sha256.New()
	_, err = h.Write(cb.Bytes())
	assert.NoError(t, err)
	hash := h.Sum(nil)

	requestPath := fmt.Sprintf("%s/%s/%s/%s-%s.json", constants.CommandName, constants.BranchName, version, runtime.GOOS, runtime.GOARCH)
	httpmock.RegisterWithResponseBody("GET", requestPath, 200, fmt.Sprintf(`{"Version": "%s", "Sha256": "%x"}`, version, hash))

	requestPath = fmt.Sprintf("%s/%s/%s/%s-%s.gz", constants.CommandName, constants.BranchName, version, runtime.GOOS, runtime.GOARCH)
	httpmock.RegisterWithResponseBytes("GET", requestPath, 200, buf.Bytes())
}

func TestForwardAndExit(t *testing.T) {
	httpmock.Activate(constants.APIUpdateURL)
	defer httpmock.DeActivate()

	setupCwd(t, true)
	mockUpdater(t, "1.2.3-123")

	var exitCode int
	exit = func(code int) {
		exitCode = code
	}

	var args = []string{"somebinary", "arg1", "arg2", "--flag"}
	forwardAndExit(args)
	assert.Equal(t, 0, exitCode, "exits with code 0")

	// Invoking the individual methods so we can capture stdout properly
	binary := forwardBin("1.2.3-123")
	out, err := osutil.CaptureStdout(func() {
		execForwardAndExit(binary, args)
	})
	assert.NoError(t, err)

	assert.Contains(t, out, fmt.Sprintf("OUTPUT--%s--OUTPUT", strings.Join(args[1:], " ")), "state.sh mock should print our args")
}

func TestForwardNotUsed(t *testing.T) {
	httpmock.Activate(constants.APIUpdateURL)
	defer httpmock.DeActivate()

	setupCwd(t, false)
	mockUpdater(t, constants.Version)

	exitCode := -1
	exit = func(code int) {
		exitCode = code
	}

	var args = []string{"somebinary", "arg1", "arg2", "--flag"}
	forwardAndExit(args)
	assert.Equal(t, -1, exitCode, "exit was not called")
}
