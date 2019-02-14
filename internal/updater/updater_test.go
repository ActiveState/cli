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

func TestUpdaterFetchMustReturnNonNilReaderCloser(t *testing.T) {
	mr := &mockRequester{}
	mr.HandleRequest(
		func(url string) (io.ReadCloser, error) {
			return nil, nil
		})
	updater := createUpdater(mr)
	err := updater.Run()
	assert.Error(t, err, "Fetch was expected to return non-nil ReadCloser")
}

func TestUpdaterWithEmptyPayloadErrorNoUpdate(t *testing.T) {
	mr := &mockRequester{}
	mr.HandleRequest(
		func(url string) (io.ReadCloser, error) {
			assert.Equal(t, "http://updates.yourdomain.com/myapp/master/"+runtime.GOOS+"-"+runtime.GOARCH+".json", url)
			return newTestReaderCloser("{}"), nil
		})
	updater := createUpdater(mr)

	err := updater.Run()
	assert.Error(t, err, "Should fail because there is no update")
}

func TestUpdaterWithEmptyPayloadNoErrorNoUpdateEscapedPath(t *testing.T) {
	var buf bytes.Buffer
	zw := gzip.NewWriter(&buf)
	zw.Name = "myapp"

	f, err := ioutil.ReadFile(os.Args[0])
	assert.NoError(t, err)
	zw.Write(f)
	zw.Close()

	mr := &mockRequester{}
	mr.HandleRequest(
		func(url string) (io.ReadCloser, error) {
			cb := &ClosingBuffer{bytes.NewBuffer(buf.Bytes())}
			h := sha256.New()
			h.Write(cb.Bytes())
			computed := h.Sum(nil)
			assert.Equal(t, "http://updates.yourdomain.com/myapp%2Bfoo/master/"+runtime.GOOS+"-"+runtime.GOARCH+".json", url)
			return newTestReaderCloser(fmt.Sprintf(`{"Version": "1.3+foobar", "Sha256": "%x"}`, computed)), nil
		})
	mr.HandleRequest(
		func(url string) (io.ReadCloser, error) {
			cb := &ClosingBuffer{bytes.NewBuffer(buf.Bytes())}
			assert.Equal(t, "http://updates.yourdomain.com/myapp%2Bfoo/master/1.3%2Bfoobar/"+runtime.GOOS+"-"+runtime.GOARCH+".gz", url)
			return cb, nil
		})
	mr.fetches = append(mr.fetches, mr.fetches[len(mr.fetches)-1]) // previous request happens twice
	updater := createUpdaterWithEscapedCharacters(mr)

	err = updater.Run()
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
	mr := &mockRequester{}
	mr.HandleRequest(
		func(url string) (io.ReadCloser, error) {
			assert.Equal(t, "http://updates.yourdomain.com/myapp/master/1.2.3-456/"+runtime.GOOS+"-"+runtime.GOARCH+".json", url)
			return newTestReaderCloser(`{"Version": "1.2.3-456", "Sha256": "9F86D081884C7D659A2FEAA0C55AD015A3BF4F1B2B0B822CD15D6C15B0F00A08"}`), nil
		})

	updater := createUpdater(mr)
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

func createUpdater(mr *mockRequester) *Updater {
	return &Updater{
		CurrentVersion: "1.2",
		APIURL:         "http://updates.yourdomain.com/",
		Dir:            "update/",
		CmdName:        "myapp", // app name
		Requester:      mr,
	}
}

func createUpdaterWithEscapedCharacters(mr *mockRequester) *Updater {
	return &Updater{
		CurrentVersion: "1.2+foobar",
		APIURL:         "http://updates.yourdomain.com/",
		Dir:            "update/",
		CmdName:        "myapp+foo", // app name
		Requester:      mr,
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
