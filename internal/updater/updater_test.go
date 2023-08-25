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
		Name            string
		Origin          *Origin
		AvailableUpdate *AvailableUpdate
		IsUseful        bool
	}{
		{
			Name:            "same-version",
			Origin:          &Origin{Channel: "master", Version: "1.2.3"},
			AvailableUpdate: newAvailableUpdate("master", "1.2.3"),
			IsUseful:        false,
		},
		{
			Name:            "updated-version",
			Origin:          &Origin{Channel: "master", Version: "2.3.4"},
			AvailableUpdate: newAvailableUpdate("master", "2.3.5"),
			IsUseful:        true,
		},
		{
			Name:            "check-different-channel",
			Origin:          &Origin{Channel: "master", Version: "3.4.5"},
			AvailableUpdate: newAvailableUpdate("beta", "3.4.5"),
			IsUseful:        true,
		},
		{
			Name:            "empty AvailableUpdate",
			Origin:          &Origin{"master", "5.6.7"},
			AvailableUpdate: &AvailableUpdate{},
			IsUseful:        false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.Name, func(t *testing.T) {
			upd := NewUpdateInstallerByOrigin(nil, tt.Origin, tt.AvailableUpdate)
			assert.Equal(t, tt.IsUseful, upd.ShouldInstall())
		})
	}
}
