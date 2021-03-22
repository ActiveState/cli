package machineid

import (
	"github.com/denisbrodbeck/machineid"
	"github.com/google/uuid"
)

type Configurable interface {
	GetString(string) string
	Set(string, interface{}) error
}

// global configuration object
var cfg Configurable

var errorLogger func(msg string, args ...interface{})

func SetConfiguration(c Configurable) {
	cfg = c
}

func SetErrorLogger(l func(msg string, args ...interface{})) {
	errorLogger = l
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
	err = cfg.Set("machineID", machineID)
	if err != nil {
		errorLogger("Could not set machineID in config, error: %v", err)
	}
	return machineID
}
