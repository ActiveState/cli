package analytics

import (
	"os"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/logging"
	ga "github.com/ActiveState/go-ogle-analytics"
	"github.com/denisbrodbeck/machineid"
)

var client *ga.Client

// CatRunCmd is the event category used for running commands
const CatRunCmd = "run-command"

func init() {
	setup()
}

func setup() {
	id, err := machineid.ID()
	if err != nil {
		logging.Error("Cannot retrieve machine ID: %s", err.Error())
		return
	}
	client, err = ga.NewClient(constants.AnalyticsTrackingID)
	if err != nil {
		logging.Error("Cannot initialize analytics: %s", err.Error())
		return
	}

	// NOTE: hostname logging is for internal use only and should be removed before we go into production
	// this is to track how we're doing on dogfooding
	hostname := "unknown"
	_hostname, err := os.Hostname()
	if err == nil {
		hostname = _hostname
		id = hostname
	} else {
		logging.Error("Cannot detect hostname: %s", err.Error())
	}

	client.ClientID(id)
	client.CustomDimensionMap(map[string]string{
		"1": hostname,
	})
}

// Event logs an event to google analytics
func Event(category string, action string) {
	go event(category, action)
}

func event(category string, action string) error {
	logging.Debug("Event: %s, %s", category, action)
	if category == CatRunCmd {
		client.Send(ga.NewPageview())
	}
	return client.Send(ga.NewEvent(category, action))
}

// EventWithValue logs an event with an integer value to google analytics
func EventWithValue(category string, action string, value int64) {
	go eventWithValue(category, action, value)
}

func eventWithValue(category string, action string, value int64) error {
	logging.Debug("Event: %s, %s", category, action)
	return client.Send(ga.NewEvent(category, action).Value(value))
}
