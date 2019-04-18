package updatemocks

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/ActiveState/archiver"
	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/testhelpers/httpmock"
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
	var archive archiver.Archiver
	var ext string

	if runtime.GOOS == "windows" {
		archive = archiver.NewZip()
		ext = ".zip"
	} else {
		archive = archiver.NewTarGz()
		ext = ".tar.gz"
	}

	tempDir, err := ioutil.TempDir("", "cli-update-mock")
	if err != nil {
		t.Fatal(fmt.Sprintf("Error creating temp dir: %v", err))
	}
	tempFile := filepath.Join(tempDir, "archive"+ext)

	err = archive.Archive([]string{filename}, tempFile)
	if err != nil {
		t.Fatal(fmt.Sprintf("Error creating temp dir: %v", err))
	}

	fileBytes, err := ioutil.ReadFile(tempFile)
	if err != nil {
		t.Fatal(fmt.Sprintf("Error reading file: %v", err))
	}

	hasher := sha256.New()
	hasher.Write(fileBytes)
	hash := hasher.Sum(nil)

	requestPath := CreateRequestPath(fmt.Sprintf("%s-%s.json", runtime.GOOS, runtime.GOARCH))
	httpmock.RegisterWithResponseBody("GET", requestPath, 200, fmt.Sprintf(`{"Version": "%s", "Sha256": "%x"}`, version, hash))

	requestPath = CreateRequestPath(fmt.Sprintf("%s/%s-%s.json", version, runtime.GOOS, runtime.GOARCH))
	httpmock.RegisterWithResponseBody("GET", requestPath, 200, fmt.Sprintf(`{"Version": "%s", "Sha256": "%x"}`, version, hash))

	requestPath = CreateRequestPath(fmt.Sprintf("%s/%s-%s%s", version, runtime.GOOS, runtime.GOARCH, ext))
	httpmock.RegisterWithResponseBytes("GET", requestPath, 200, fileBytes)
}
