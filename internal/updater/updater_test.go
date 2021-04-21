package updater

import (
	"encoding/json"
	"fmt"
	"runtime"
	"testing"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type httpGetMock struct {
	UrlCalled      string
	MockedResponse []byte
}

func (m *httpGetMock) Get(url string) ([]byte, error) {
	m.UrlCalled = url
	return m.MockedResponse, nil
}

func mockUpdate(channel, version string) *AvailableUpdate {
	return NewAvailableUpdate(version, channel, "platform", "path/to/zipfile.zip", "123456")
}

func newMock(t *testing.T, channel, version string) *httpGetMock {
	up := mockUpdate(channel, version)

	b, err := json.Marshal(up)
	require.NoError(t, err)
	return &httpGetMock{MockedResponse: b}
}

func expectedUrl(infix string) string {
	platform := runtime.GOOS + "-" + runtime.GOARCH
	return fmt.Sprintf("https://state-tool.s3.amazonaws.com/update/%s/%s/info.json", infix, platform)
}

func TestCheckerCheckFor(t *testing.T) {
	tests := []struct {
		Name           string
		MockChannel    string
		MockVersion    string
		CheckChannel   string
		CheckVersion   string
		ExpectedResult *AvailableUpdate
		ExpectedUrl    string
	}{
		{
			Name:        "same-version",
			MockChannel: "master", MockVersion: "1.2.3",
			CheckChannel: "", CheckVersion: "",
			ExpectedResult: nil,
			ExpectedUrl:    expectedUrl("master"),
		},
		{
			Name:        "updated-version",
			MockChannel: "master", MockVersion: "2.3.4",
			CheckChannel: "", CheckVersion: "",
			ExpectedResult: mockUpdate("master", "2.3.4"),
			ExpectedUrl:    expectedUrl("master"),
		},
		{
			Name:        "check-different-channel",
			MockChannel: "release", MockVersion: "1.2.3",
			CheckChannel: "release", CheckVersion: "",
			ExpectedResult: mockUpdate("release", "1.2.3"),
			ExpectedUrl:    expectedUrl("release"),
		},
		{
			Name:        "specific-version",
			MockChannel: "master", MockVersion: "0.1.2",
			CheckChannel: "master", CheckVersion: "0.1.2",
			ExpectedResult: mockUpdate("master", "0.1.2"),
			ExpectedUrl:    expectedUrl("master/0.1.2"),
		},
		{
			Name:        "check-same-version",
			MockChannel: "master", MockVersion: "1.2.3",
			CheckChannel: "master", CheckVersion: "1.2.3",
			ExpectedResult: nil,
			ExpectedUrl:    expectedUrl("master/1.2.3"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.Name, func(t *testing.T) {
			m := newMock(t, tt.MockChannel, tt.MockVersion)
			check := NewChecker(constants.APIUpdateURL, "master", "1.2.3", m)
			res, err := check.CheckFor(tt.CheckChannel, tt.CheckVersion)
			require.NoError(t, err)
			if res != nil {
				res.url = ""
			}
			assert.Equal(t, tt.ExpectedResult, res)
			assert.Equal(t, tt.ExpectedUrl, m.UrlCalled)
		})
	}
}
