package updater

import (
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"testing"

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
			assert.Equal(t, "http://updates.yourdomain.com/myapp/linux-amd64.json", url)
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

	var cb *ClosingBuffer
	cb = &ClosingBuffer{bytes.NewBuffer(buf.Bytes())}

	mr := &mockRequester{}
	mr.HandleRequest(
		func(url string) (io.ReadCloser, error) {
			h := sha256.New()
			h.Write(cb.Bytes())
			computed := h.Sum(nil)
			assert.Equal(t, "http://updates.yourdomain.com/myapp%2Bfoo/linux-amd64.json", url)
			return newTestReaderCloser(fmt.Sprintf(`{"Version": "1.3+foobar", "Sha256": "%x"}`, computed)), nil
		})
	mr.HandleRequest(
		func(url string) (io.ReadCloser, error) {
			assert.Equal(t, "http://updates.yourdomain.com/myapp%2Bfoo/1.3%2Bfoobar/linux-amd64.gz", url)
			return cb, nil
		})
	updater := createUpdaterWithEscapedCharacters(mr)

	err = updater.Run()
	assert.NoError(t, err, "Should run update")
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
