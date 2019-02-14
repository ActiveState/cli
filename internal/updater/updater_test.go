package updater

import (
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/locale"
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

var testHash = sha256.New()

func createRequestPath(append string) string {
	return fmt.Sprintf("%s/%s/%s", constants.CommandName, constants.BranchName, append)
}

func mockUpdater(t *testing.T, version string) {
	var buf bytes.Buffer
	zw := gzip.NewWriter(&buf)
	zw.Name = "myapp"

	f, err := ioutil.ReadFile(os.Args[0])
	assert.NoError(t, err)
	_, err = zw.Write(f)
	assert.NoError(t, err)
	assert.NoError(t, zw.Close())

	cb := &ClosingBuffer{bytes.NewBuffer(buf.Bytes())}
	h := sha256.New()
	_, err = h.Write(cb.Bytes())
	assert.NoError(t, err)
	hash := h.Sum(nil)

	requestPath := createRequestPath(fmt.Sprintf("%s-%s.json", runtime.GOOS, runtime.GOARCH))
	httpmock.RegisterWithResponseBody("GET", requestPath, 200, fmt.Sprintf(`{"Version": "%s", "Sha256": "%x"}`, version, hash))

	requestPath = createRequestPath(fmt.Sprintf("%s/%s-%s.json", version, runtime.GOOS, runtime.GOARCH))
	httpmock.RegisterWithResponseBody("GET", requestPath, 200, fmt.Sprintf(`{"Version": "%s", "Sha256": "%x"}`, version, hash))

	requestPath = createRequestPath(fmt.Sprintf("%s/%s-%s.gz", version, runtime.GOOS, runtime.GOARCH))
	httpmock.RegisterWithResponseBytes("GET", requestPath, 200, buf.Bytes())
}

func TestUpdaterWithEmptyPayloadErrorNoUpdate(t *testing.T) {
	httpmock.Activate(constants.APIUpdateURL)
	defer httpmock.DeActivate()
	httpmock.RegisterWithResponseBody("GET", createRequestPath(fmt.Sprintf("%s-%s.json", runtime.GOOS, runtime.GOARCH)), 200, "{}")

	updater := createUpdater()

	err := updater.Run()
	assert.Error(t, err, "Should fail because there is no update")
}

func TestUpdaterNoError(t *testing.T) {
	httpmock.Activate(constants.APIUpdateURL)
	defer httpmock.DeActivate()

	mockUpdater(t, "1.3")

	updater := createUpdater()

	err := updater.Run()
	assert.NoError(t, err, "Should run update")

	dir, err := ioutil.TempDir("", "state-test-updater")
	assert.NoError(t, err)
	target := filepath.Join(dir, "target")
	if fileutils.FileExists(target) {
		os.Remove(target)
	}

	err = updater.Download(target)
	assert.NoError(t, err)
	assert.FileExists(t, target, "Downloads to target path")

	os.Remove(target)
}

func TestUpdaterInfoDesiredVersion(t *testing.T) {
	httpmock.Activate(constants.APIUpdateURL)
	defer httpmock.DeActivate()
	httpmock.RegisterWithResponseBody(
		"GET",
		createRequestPath(fmt.Sprintf("1.2.3-456/%s-%s.json", runtime.GOOS, runtime.GOARCH)),
		200,
		`{"Version": "1.2.3-456", "Sha256": "9F86D081884C7D659A2FEAA0C55AD015A3BF4F1B2B0B822CD15D6C15B0F00A08"}`)

	updater := createUpdater()
	updater.DesiredVersion = "1.2.3-456"
	info, err := updater.Info()
	assert.NoError(t, err)

	assert.NotNil(t, info, "Returns update info")
	assert.Equal(t, "1.2.3-456", info.Version, "Should return expected version")
}

func TestPrintUpdateMessage(t *testing.T) {
	setup(t, true)

	httpmock.Activate(constants.APIUpdateURL)
	defer httpmock.DeActivate()

	requestPath := fmt.Sprintf("%s/%s/%s-%s.json", constants.CommandName, constants.BranchName, runtime.GOOS, runtime.GOARCH)
	httpmock.RegisterWithResponseBody("GET", requestPath, 200, `{"Version": "1.2.3-456", "Sha256": "9F86D081884C7D659A2FEAA0C55AD015A3BF4F1B2B0B822CD15D6C15B0F00A08"}`)

	stdout, err := osutil.CaptureStdout(func() {
		PrintUpdateMessage()
	})
	assert.NoError(t, err)

	assert.Contains(t, stdout, locale.Tr("update_available", constants.Version, "1.2.3-456"), "Should print an update message")
}

func TestPrintUpdateMessageEmpty(t *testing.T) {
	setup(t, false)

	stdout, err := osutil.CaptureStdout(func() {
		PrintUpdateMessage()
	})
	assert.NoError(t, err)
	assert.Empty(t, stdout, "Should not print an update message because the version is not locked")
}

func createUpdater() *Updater {
	return &Updater{
		CurrentVersion: "1.2",
		APIURL:         constants.APIUpdateURL,
		Dir:            constants.UpdateStorageDir,
		CmdName:        constants.CommandName, // app name
	}
}

type testReadCloser struct {
	buffer *bytes.Buffer
}

func newTestReaderCloser(payload string) io.ReadCloser {
	return &testReadCloser{buffer: bytes.NewBufferString(payload)}
}

func (trc *testReadCloser) Read(p []byte) (n int, err error) {
	return trc.buffer.Read(p)
}

func (trc *testReadCloser) Close() error {
	return nil
}
