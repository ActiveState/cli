package updater

import (
	"bytes"
	"fmt"
	"io"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/environment"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/testhelpers/httpmock"
	"github.com/ActiveState/cli/internal/testhelpers/outputhelper"
	"github.com/ActiveState/cli/internal/testhelpers/updatemocks"
)

var configPath = filepath.Join(environment.GetRootPathUnsafe(), "internal", "updater", "testdata", constants.ConfigFileName)
var configPathWithVersion = filepath.Join(environment.GetRootPathUnsafe(), "internal", "updater", "testdata", "withversion", constants.ConfigFileName)

func TestUpdaterWithEmptyPayloadErrorNoUpdate(t *testing.T) {
	httpmock.Activate(constants.APIUpdateURL)
	defer httpmock.DeActivate()
	httpmock.RegisterWithResponseBody("GET", updatemocks.CreateRequestPath(constants.BranchName, fmt.Sprintf("%s-%s.json", runtime.GOOS, runtime.GOARCH)), 200, "{}")

	updater := createUpdater()

	out := outputhelper.NewCatcher()
	err := updater.Run(out.Outputer)
	assert.Error(t, err, "Should fail because there is no update")
	assert.Equal(t, "", strings.TrimSpace(out.CombinedOutput()))
}

func TestUpdaterInfoDesiredVersion(t *testing.T) {
	httpmock.Activate(constants.APIUpdateURL)
	defer httpmock.DeActivate()
	httpmock.RegisterWithResponseBody(
		"GET",
		updatemocks.CreateRequestPath(constants.BranchName, fmt.Sprintf("1.2.3-456/%s-%s.json", runtime.GOOS, runtime.GOARCH)),
		200,
		`{"Version": "1.2.3-456", "Sha256v2": "9F86D081884C7D659A2FEAA0C55AD015A3BF4F1B2B0B822CD15D6C15B0F00A08"}`)

	updater := createUpdater()
	updater.DesiredVersion = "1.2.3-456"
	info, err := updater.Info()
	require.NoError(t, err)

	assert.NotNil(t, info, "Returns update info")
	assert.Equal(t, "1.2.3-456", info.Version, "Should return expected version")
}

func TestPrintUpdateMessage(t *testing.T) {
	setup(t, true)

	httpmock.Activate(constants.APIUpdateURL)
	defer httpmock.DeActivate()

	requestPath := fmt.Sprintf("%s/%s/%s-%s.json", constants.CommandName, constants.BranchName, runtime.GOOS, runtime.GOARCH)
	httpmock.RegisterWithResponseBody("GET", requestPath, 200, `{"Version": "1.2.3-456", "Sha256v2": "9F86D081884C7D659A2FEAA0C55AD015A3BF4F1B2B0B822CD15D6C15B0F00A08"}`)

	out := outputhelper.NewCatcher()
	PrintUpdateMessage(configPathWithVersion, out)

	assert.Contains(t, out.CombinedOutput(), output.StripColorCodes(locale.Tr("update_available", constants.Version, "1.2.3-456")), "Should print an update message")
}

func TestPrintUpdateMessageEmpty(t *testing.T) {
	setup(t, false)

	out := outputhelper.NewCatcher()
	PrintUpdateMessage(configPath, out)

	assert.Empty(t, out.ErrorOutput(), "Should not print an update message because the version is not locked")
}

func createUpdater() *Updater {
	return &Updater{
		CurrentVersion: "1.2",
		APIURL:         constants.APIUpdateURL,
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
