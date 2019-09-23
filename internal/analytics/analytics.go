package analytics

import (
	"github.com/denisbrodbeck/machineid"

	"github.com/ActiveState/cli/internal/condition"
	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/logging"
	ga "github.com/ActiveState/go-ogle-analytics"
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
		id = "unknown"
	}
	client, err = ga.NewClient(constants.AnalyticsTrackingID)
	if err != nil {
		logging.Error("Cannot initialize analytics: %s", err.Error())
		return
	}

	client.ClientID(id)
	client.CustomDimensionMap(map[string]string{
		// Commented out idx 1 so it's clear why we start with 2. We used to log the hostname while dogfooding internally.
		// "1": "hostname (deprected)"
		"2": constants.Version,
		"3": constants.BranchName,
	})

	if id == "unknown" {
		Event("error", "unknown machine id")
	}
}

// Event logs an event to google analytics
func Event(category string, action string) {
	go event(category, action)
}

func event(category string, action string) error {
	if client == nil || condition.InTest() {
		return nil
	}

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
	if client == nil || condition.InTest() {
		return nil
	}

	logging.Debug("Event: %s, %s", category, action)
	return client.Send(ga.NewEvent(category, action).Value(value))
}
