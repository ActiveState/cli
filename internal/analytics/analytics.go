package analytics

import (
	"fmt"
	"net/http"
	"net/url"
	"sync"

	ga "github.com/ActiveState/go-ogle-analytics"
	"github.com/ActiveState/sysinfo"

	"github.com/ActiveState/cli/internal/condition"
	"github.com/ActiveState/cli/internal/config"
	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/machineid"
	"github.com/ActiveState/cli/internal/output"
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

// CatRuntime is the event category used for all runtime setup and usage
const CatRuntime = "runtime"

// ActRuntimeStart is the event action sent when creating a runtime
const ActRuntimeStart = "start"

// ActRuntimeCache is the event action sent when a runtime is constructed from the local cache alone
const ActRuntimeCache = "cache"

// ActRuntimeBuild is the event action sent when starting a remote build for the project
const ActRuntimeBuild = "build"

// ActRuntimeDownload is the event action sent before starting the download of artifacts for a runtime
const ActRuntimeDownload = "download"

// ActRuntimeSuccess is the event action sent when a runtime's environment has been successfully computed (for the first time)
const ActRuntimeSuccess = "success"

// ActRuntimeFailure is the event action sent when a failure occurred anytime during a runtime operation
const ActRuntimeFailure = "failure"

// LblRtFailUpdate is the label sent with an ActRuntimeFailure event if an error occurred during a runtime update
const LblRtFailUpdate = "update"

// LblRtFailEnv is the label sent with  an ActRuntimeFailure event if an error occurred during the resolution of the runtime environment
const LblRtFailEnv = "env"

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

func Wait() {
	eventWaitGroup.Wait()
}

func setup() {
	installSource := "unknown-due-to-config-error"
	cfg, err := config.Get()
	if err != nil {
		logging.Error("Could not detect installSource: %s", errs.Join(err, " :: ").Error())
	} else {
		installSource = cfg.InstallSource()
	}

	id := machineid.UniqID()
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
		installSource: installSource,
		machineID:     machineid.UniqID(),
		output:        string(output.PlainFormatName),
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
	logging.Error("Error during analytics.sendEvent: %v", errs.Join(err, ":"))
}

func sendEvent(category, action, label string, dimensions map[string]string) error {
	if deferAnalytics {
		// TODO: figure out a way to pass configuration
		cfg, err := config.Get()
		if err != nil {
			return locale.WrapError(err, "config_get_error")
		}
		if err := deferEvent(cfg, category, action, label, dimensions); err != nil {
			return locale.WrapError(err, "err_analytics_defer", "Could not defer event")
		}
		return nil
	}

	eventWaitGroup.Add(2)
	go sendGAEvent(category, action, label, dimensions)
	go sendS3Pixel(category, action, label, dimensions)

	return nil
}

func sendGAEvent(category, action, label string, dimensions map[string]string) {
	defer eventWaitGroup.Done()
	logging.Debug("Sending Google Analytics event with: %s, %s, %s", category, action, label)

	if client == nil {
		logging.Error("Client is not set")
		return
	}
	client.CustomDimensionMap(dimensions)

	if category == CatRunCmd {
		client.Send(ga.NewPageview())
	}
	event := ga.NewEvent(category, action)
	if label != "" {
		event.Label(label)
	}
	err := client.Send(event)
	if err != nil {
		logging.Error("Could not send GA Event: %v", err)
	}
}

func sendS3Pixel(category, action, label string, dimensions map[string]string) {
	defer eventWaitGroup.Done()
	logging.Debug("Sending S3 pixel event with: %s, %s, %s", category, action, label)
	pixelURL, err := url.Parse("https://state-tool.s3.amazonaws.com/pixel")
	if err != nil {
		logging.Error("Invalid URL for analytics S3 pixel")
		return
	}

	query := pixelURL.Query()
	query.Add("x-category", category)
	query.Add("x-action", action)
	query.Add("x-label", label)

	for num, value := range dimensions {
		key := fmt.Sprintf("x-custom%s", num)
		query.Add(key, value)
	}
	pixelURL.RawQuery = query.Encode()

	logging.Debug("Using S3 pixel URL: ", pixelURL.String())
	_, err = http.Head(pixelURL.String())
	if err != nil {
		logging.Error("Could not download S3 pixel: %v", err)
		return
	}
}
