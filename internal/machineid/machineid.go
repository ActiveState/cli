package machineid

import (
	"github.com/denisbrodbeck/machineid"
	"github.com/google/uuid"
)

const FallbackID = "99999999-9999-9999-9999-999999999999"

type Configurable interface {
	GetString(string) string
	Set(string, interface{}) error
}

// global configuration object
var errorLogger func(msg string, args ...interface{})

var id *string

func Configure(c Configurable) {
	_id := UniqIDCustom(machineid.ID, func() string { return uuid.New().String() }, c)
	id = &_id
}

func SetErrorLogger(l func(msg string, args ...interface{})) {
	errorLogger = l
}

// UniqID returns a unique ID for the current platform
func UniqID() string {
	if id == nil {
		// We do not log here, as it may create a recursion
		return FallbackID
	}
	return *id
}

func UniqIDCustom(machineIDGetter func() (string, error), uuidGetter func() string, c Configurable) string {
	machID, err := machineIDGetter()
	if err == nil {
		return machID
	}

	if c == nil {
		// We do not log here, as it may create a recursion
		return "11111111-1111-1111-1111-111111111111"
	}

	machineID := c.GetString("machineID")
	if machineID != "" {
		return machineID
	}

	machineID = uuidGetter()
	err = c.Set("machineID", machineID)
	if err != nil {
		errorLogger("Could not set machineID in config, error: %v", err)
	}
	return machineID
}
