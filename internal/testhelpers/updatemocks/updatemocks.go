package updatemocks

import (
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"fmt"
	"io/ioutil"
	"runtime"
	"testing"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/testhelpers/httpmock"
	"github.com/stretchr/testify/assert"
)

type ClosingBuffer struct {
	*bytes.Buffer
}

func (cb *ClosingBuffer) Close() (err error) {
	return
}

func CreateRequestPath(append string) string {
	return fmt.Sprintf("%s/%s/%s", constants.CommandName, constants.BranchName, append)
}

// MockUpdater fully mocks an update, so that you could run the update logic and it doesn't fail
func MockUpdater(t *testing.T, filename string, version string) {
	var buf bytes.Buffer
	zw := gzip.NewWriter(&buf)
	zw.Name = constants.CommandName

	f, err := ioutil.ReadFile(filename)
	assert.NoError(t, err)
	_, err = zw.Write(f)
	assert.NoError(t, err)
	assert.NoError(t, zw.Close())

	cb := &ClosingBuffer{bytes.NewBuffer(buf.Bytes())}
	h := sha256.New()
	_, err = h.Write(cb.Bytes())
	assert.NoError(t, err)
	hash := h.Sum(nil)

	requestPath := CreateRequestPath(fmt.Sprintf("%s-%s.json", runtime.GOOS, runtime.GOARCH))
	httpmock.RegisterWithResponseBody("GET", requestPath, 200, fmt.Sprintf(`{"Version": "%s", "Sha256": "%x"}`, version, hash))

	requestPath = CreateRequestPath(fmt.Sprintf("%s/%s-%s.json", version, runtime.GOOS, runtime.GOARCH))
	httpmock.RegisterWithResponseBody("GET", requestPath, 200, fmt.Sprintf(`{"Version": "%s", "Sha256": "%x"}`, version, hash))

	requestPath = CreateRequestPath(fmt.Sprintf("%s/%s-%s.gz", version, runtime.GOOS, runtime.GOARCH))
	httpmock.RegisterWithResponseBytes("GET", requestPath, 200, buf.Bytes())
}
