package machineid

import (
	"github.com/ActiveState/cli/internal/config"
	"github.com/denisbrodbeck/machineid"
	"github.com/google/uuid"
)

// UniqID returns a unique ID for the current platform
func UniqID() string {
	return uniqID(machineid.ID, func() string { return uuid.New().String() })
}

func uniqID(machineIDGetter func() (string, error), uuidGetter func() string) string {
	machID, err := machineIDGetter()
	if err == nil {
		return machID
	}

	machineID := config.Get().GetString("machineID")
	if machineID != "" {
		return machineID
	}

	machineID = uuidGetter()
	config.Get().Set("machineID", machineID)
	return machineID
}
