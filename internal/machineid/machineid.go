package machineid

import (
	"github.com/denisbrodbeck/machineid"
	"github.com/google/uuid"
)

type Configurable interface {
	GetString(string) string
	Set(string, interface{})
}

// global configuration object
var cfg Configurable

func SetConfiguration(c Configurable) {
	cfg = c
}

// UniqID returns a unique ID for the current platform
func UniqID() string {
	return uniqID(machineid.ID, func() string { return uuid.New().String() })
}

func uniqID(machineIDGetter func() (string, error), uuidGetter func() string) string {
	machID, err := machineIDGetter()
	if err == nil {
		return machID
	}

	if cfg == nil {
		// We do not log here, as it may create a recursion
		return "11111111-1111-1111-1111-111111111111"
	}

	machineID := cfg.GetString("machineID")
	if machineID != "" {
		return machineID
	}

	machineID = uuidGetter()
	cfg.Set("machineID", machineID)
	return machineID
}
