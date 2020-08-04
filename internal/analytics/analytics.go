package analytics

import (
	"fmt"

	ga "github.com/ActiveState/go-ogle-analytics"
	"github.com/ActiveState/sysinfo"

	"github.com/ActiveState/cli/internal/condition"
	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/pkg/platform/authentication"
)

var client *ga.Client

// CustomDimensions represents the custom dimensions sent with each event
var CustomDimensions *customDimensions

// CatRunCmd is the event category used for running commands
const CatRunCmd = "run-command"

// CatBuild is the event category used for headchef builds
const CatBuild = "build"

// ActBuildProject is the event action for requesting a build for a specific project
const ActBuildProject = "project"

// CatTutorial is the event category used for tutorial level events
const CatTutorial = "tutorial"

type customDimensions struct {
	version    string
	branchName string
	userID     string
	output     string
	osName     string
	osVersion  string
}

func (d *customDimensions) SetOutput(output string) {
	d.output = output
}

func (d *customDimensions) toMap() map[string]string {
	return map[string]string{
		// Commented out idx 1 so it's clear why we start with 2. We used to log the hostname while dogfooding internally.
		// "1": "hostname (deprected)"
		"2": d.version,
		"3": d.branchName,
		"4": d.userID,
		"5": d.output,
		"6": d.osName,
		"7": d.osVersion,
	}
}

func init() {
	setup()
}

func setup() {
	id := logging.UniqID()
	var err error
	client, err = ga.NewClient(constants.AnalyticsTrackingID)
	if err != nil {
		logging.Error("Cannot initialize analytics: %s", err.Error())
		return
	}

	var userIDString string
	userID := authentication.Get().UserID()
	if userID != nil {
		userIDString = userID.String()
	}

	osName := sysinfo.OS().String()
	osVersion := "unknown"
	osvInfo, err := sysinfo.OSVersion()
	if err != nil {
		logging.Errorf("Could not detect osVersion: %v", err)
	}
	if osvInfo != nil {
		osVersion = osvInfo.Version
	}
	if osVersion == "unknown" {
		logging.SendToRollbarWhenReady("warning", fmt.Sprintf("Cannot detect the OS version: %v", err))
	}

	CustomDimensions = &customDimensions{
		version:    constants.Version,
		branchName: constants.BranchName,
		userID:     userIDString,
		osName:     osName,
		osVersion:  osVersion,
	}

	client.ClientID(id)

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
	client.CustomDimensionMap(CustomDimensions.toMap())

	logging.Debug("Event: %s, %s", category, action)
	if category == CatRunCmd {
		client.Send(ga.NewPageview())
	}
	return client.Send(ga.NewEvent(category, action))
}

// EventWithLabel logs an event with a label to google analytics
func EventWithLabel(category string, action string, label string) {
	go eventWithLabel(category, action, label)
}

func eventWithLabel(category, action, label string) error {
	if client == nil || condition.InTest() {
		return nil
	}
	client.CustomDimensionMap(CustomDimensions.toMap())

	logging.Debug("Event+label: %s, %s, %s", category, action, label)
	return client.Send(ga.NewEvent(category, action).Label(label))
}

// EventWithValue logs an event with an integer value to google analytics
func EventWithValue(category string, action string, value int64) {
	go eventWithValue(category, action, value)
}

func eventWithValue(category string, action string, value int64) error {
	if client == nil || condition.InTest() {
		return nil
	}
	client.CustomDimensionMap(CustomDimensions.toMap())

	logging.Debug("Event+value: %s, %s, %s", category, action, value)
	return client.Send(ga.NewEvent(category, action).Value(value))
}
