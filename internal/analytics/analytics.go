package analytics

import (
	"fmt"
	"sync"
	"time"

	ga "github.com/ActiveState/go-ogle-analytics"
	"github.com/ActiveState/sysinfo"
	"github.com/spf13/viper"

	"github.com/ActiveState/cli/internal/condition"
	"github.com/ActiveState/cli/internal/config"
	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/machineid"
	"github.com/ActiveState/cli/pkg/platform/authentication"
	"github.com/ActiveState/cli/pkg/projectfile"
)

var client *ga.Client

// CustomDimensions represents the custom dimensions sent with each event
var CustomDimensions *customDimensions

// CatRunCmd is the event category used for running commands
const CatRunCmd = "run-command"

// CatBuild is the event category used for headchef builds
const CatBuild = "build"

// CatPpmConversion is the event category used for ppm-conversion events
const CatPpmConversion = "ppm-conversion"

// ActBuildProject is the event action for requesting a build for a specific project
const ActBuildProject = "project"

// CatPPMShimCmd is the event category used for PPM shim events
const CatPPMShimCmd = "ppm-shim"

// CatTutorial is the event category used for tutorial level events
const CatTutorial = "tutorial"

// CatCommandExit is the event category used to track the success of state commands
const CatCommandExit = "command-exit"

// CatActivationFlow is for events that outline the activation flow
const CatActivationFlow = "activation"

// CatPrompt is for prompt events
const CatPrompt = "prompt"

type customDimensions struct {
	version       string
	branchName    string
	userID        string
	output        string
	osName        string
	osVersion     string
	installSource string
	machineID     string
	projectName   string
}

func (d *customDimensions) SetOutput(output string) {
	d.output = output
}

func (d *customDimensions) toMap() map[string]string {
	pj := projectfile.GetPersisted()
	d.projectName = ""
	if pj != nil {
		d.projectName = pj.Owner() + "/" + pj.Name()
	}
	return map[string]string{
		// Commented out idx 1 so it's clear why we start with 2. We used to log the hostname while dogfooding internally.
		// "1": "hostname (deprected)"
		"2":  d.version,
		"3":  d.branchName,
		"4":  d.userID,
		"5":  d.output,
		"6":  d.osName,
		"7":  d.osVersion,
		"8":  d.installSource,
		"9":  d.machineID,
		"10": d.projectName,
	}
}

var (
	eventWaitGroup sync.WaitGroup
)

func init() {
	CustomDimensions = &customDimensions{}
	setup()
}

// WaitForAllEvents waits for all events to return
func WaitForAllEvents(t time.Duration) {
	wg := make(chan struct{})
	go func() {
		eventWaitGroup.Wait()
		close(wg)
	}()

	select {
	case <-time.After(t):
		return
	case <-wg:
		return
	}
}

func setup() {
	id := machineid.UniqID()
	var err error
	var trackingID string
	if !condition.InTest() {
		trackingID = constants.AnalyticsTrackingID
	}

	client, err = ga.NewClient(trackingID)
	if err != nil {
		logging.Error("Cannot initialize analytics: %s", err.Error())
		client = nil
	}

	if client != nil {
		client.ClientID(id)
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
		version:       constants.Version,
		branchName:    constants.BranchName,
		userID:        userIDString,
		osName:        osName,
		osVersion:     osVersion,
		installSource: config.InstallSource(),
		machineID:     machineid.UniqID(),
	}

	if id == "unknown" {
		logging.Error("unknown machine id")
	}
}

// Event logs an event to google analytics
func Event(category string, action string) {
	eventWaitGroup.Add(1)
	go func() {
		defer eventWaitGroup.Done()
		event(category, action)
	}()
}

func event(category string, action string) {
	sendEventAndLog(category, action, "", CustomDimensions.toMap())
}

// EventWithLabel logs an event with a label to google analytics
func EventWithLabel(category string, action string, label string) {
	eventWaitGroup.Add(1)
	go func() {
		defer eventWaitGroup.Done()
		eventWithLabel(category, action, label)
	}()
}

func eventWithLabel(category, action, label string) {
	sendEventAndLog(category, action, label, CustomDimensions.toMap())
}

func sendEventAndLog(category, action, label string, dimensions map[string]string) {
	err := sendEvent(category, action, label, dimensions)
	if err == nil {
		return
	}
	logging.Error("Error during analytics.sendEvent: %v", err)
}

func sendEvent(category, action, label string, dimensions map[string]string) error {
	if deferAnalytics {
		if err := deferEvent(category, action, label, dimensions); err != nil {
			return locale.WrapError(err, "err_analytics_defer", "Could not defer event")
		}
		if err := viper.WriteConfig(); err != nil { // the global viper instance is bugged, need to work around it for now -- https://www.pivotaltracker.com/story/show/175624789
			return locale.WrapError(err, "err_viper_write_defer", "Could not save configuration on defer")
		}
	}

	logging.Debug("Sending: %s, %s, %s", category, action, label)

	if client == nil {
		logging.Error("Client is not set")
		return nil
	}
	client.CustomDimensionMap(dimensions)

	if category == CatRunCmd {
		client.Send(ga.NewPageview())
	}
	event := ga.NewEvent(category, action)
	if label != "" {
		event.Label(label)
	}
	return client.Send(event)
}
