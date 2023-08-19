package updater

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func newAvailableUpdate(channel, version string) *AvailableUpdate {
	return NewAvailableUpdate(channel, version, "platform", "path/to/zipfile.zip", "123456", "")
}

func TestUpdateNotNeeded(t *testing.T) {
	tests := []struct {
		Name             string
		OriginChannel    string
		OriginVersion    string
		AvailableChannel string
		AvailableVersion string
		NotNeeded        bool
	}{
		{
			Name:          "same-version",
			OriginChannel: "master", OriginVersion: "1.2.3",
			AvailableChannel: "master", AvailableVersion: "1.2.3",
			NotNeeded: true,
		},
		{
			Name:          "updated-version",
			OriginChannel: "master", OriginVersion: "2.3.4",
			AvailableChannel: "master", AvailableVersion: "2.3.5",
			NotNeeded: false,
		},
		{
			Name:          "check-different-channel",
			OriginChannel: "master", OriginVersion: "1.2.3",
			AvailableChannel: "beta", AvailableVersion: "1.2.3",
			NotNeeded: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.Name, func(t *testing.T) {
			upd := NewUpdateByOrigin(nil, &Origin{Channel: tt.OriginChannel, Version: tt.OriginVersion}, newAvailableUpdate(tt.AvailableChannel, tt.AvailableVersion))
			assert.Equal(t, tt.NotNeeded, upd.NotNeeded())
		})
	}
}
