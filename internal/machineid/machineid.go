package machineid

import (
	"github.com/denisbrodbeck/machineid"
	"github.com/google/uuid"
	"github.com/spf13/viper"
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

	machineID := viper.GetString("machineID")
	if machineID != "" {
		return machineID
	}

	machineID = uuidGetter()
	viper.Set("machineID", machineID)
	return machineID
}
