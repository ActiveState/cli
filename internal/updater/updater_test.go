package updater

import (
	"encoding/json"
	"testing"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type httpGetMock struct {
	UrlCalled      string
	MockedResponse []byte
}

func (m *httpGetMock) Get(url string) ([]byte, int, error) {
	m.UrlCalled = url
	return m.MockedResponse, 0, nil
}

func mockUpdate(channel, version, tag string) *AvailableUpdate {
	return NewAvailableUpdate(version, channel, "platform", "path/to/zipfile.zip", "123456", tag)
}

func newMock(t *testing.T, channel, version, tag string) *httpGetMock {
	up := mockUpdate(channel, version, tag)

	b, err := json.Marshal(up)
	require.NoError(t, err)
	return &httpGetMock{MockedResponse: b}
}

type configMock struct{}

func (cm *configMock) GetString(string) string {
	return ""
}

func TestCheckerCheckFor(t *testing.T) {
	tests := []struct {
		Name           string
		MockChannel    string
		MockVersion    string
		MockTag        string
		CheckChannel   string
		CheckVersion   string
		ExpectedResult *AvailableUpdate
	}{
		{
			Name:        "same-version",
			MockChannel: "master", MockVersion: "1.2.3",
			CheckChannel: "", CheckVersion: "",
			ExpectedResult: nil,
		},
		{
			Name:        "updated-version",
			MockChannel: "master", MockVersion: "2.3.4",
			CheckChannel: "", CheckVersion: "",
			ExpectedResult: mockUpdate("master", "2.3.4", ""),
		},
		{
			Name:        "check-different-channel",
			MockChannel: "release", MockVersion: "1.2.3", MockTag: "experiment",
			CheckChannel: "release", CheckVersion: "",
			ExpectedResult: mockUpdate("release", "1.2.3", "experiment"),
		},
		{
			Name:        "specific-version",
			MockChannel: "master", MockVersion: "0.1.2",
			CheckChannel: "master", CheckVersion: "0.1.2",
			ExpectedResult: mockUpdate("master", "0.1.2", ""),
		},
		{
			Name:        "check-same-version",
			MockChannel: "master", MockVersion: "1.2.3",
			CheckChannel: "master", CheckVersion: "1.2.3",
			ExpectedResult: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.Name, func(t *testing.T) {
			m := newMock(t, tt.MockChannel, tt.MockVersion, tt.MockTag)
			check := NewChecker(&configMock{}, constants.APIUpdateInfoURL, constants.APIUpdateURL, "master", "1.2.3", m)
			res, err := check.CheckFor(tt.CheckChannel, tt.CheckVersion)
			require.NoError(t, err)
			if res != nil {
				res.url = ""
			}
			assert.Equal(t, tt.ExpectedResult, res)
		})
	}
}
