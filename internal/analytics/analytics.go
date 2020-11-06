package analytics

import (
	"fmt"
	"sync"

	"github.com/ActiveState/sysinfo"
	"github.com/go-openapi/strfmt"

	"github.com/ActiveState/cli/internal/condition"
	"github.com/ActiveState/cli/internal/config"
	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/loghttp"
	"github.com/ActiveState/cli/pkg/platform/authentication"
	ga "github.com/ActiveState/go-ogle-analytics"
)

const (
	// CatRunCmd is the event category used for running commands
	CatRunCmd = "run-command"
	// CatBuild is the event category used for headchef builds
	CatBuild = "build"
	// CatPpmConversion is the event category used for ppm-conversion events
	CatPpmConversion = "ppm-conversion"
	// ActBuildProject is the event action for requesting a build for a specific project
	ActBuildProject = "project"
	// CatPPMShimCmd is the event category used for PPM shim events
	CatPPMShimCmd = "ppm-shim"
	// CatTutorial is the event category used for tutorial level events
	CatTutorial = "tutorial"
	// CatCommandExit is the event category used to track the success of state commands
	CatCommandExit = "command-exit"
	// CatActivationFlow is for events that outline the activation flow
	CatActivationFlow = "activation"

	// ValUnknown is a token used to indicate an unknown value
	ValUnknown = "unknown"
)

type Analytics struct {
	client       *client
	UniqClientID string
	Dimensions   *CustomDimensions
	VersInfoErr  error
	wg           sync.WaitGroup
}

func NewAnalytics(logFn loghttp.LogFunc, uniqID string, userID *strfmt.UUID) (*Analytics, error) {
	client, err := newClient(logFn, uniqID)
	if err != nil {
		return nil, errs.Wrap(err, "Cannot create new analytics client")
	}

	var userIDString string
	if userID != nil {
		userIDString = userID.String()
	}

	osName := sysinfo.OS().String()
	osVersion := ValUnknown
	var versInfoErr error
	osvInfo, err := sysinfo.OSVersion()
	if err != nil {
		versInfoErr = fmt.Errorf("Could not detect osVersion: %w", err)
	}
	if osvInfo != nil {
		osVersion = osvInfo.Version
	}

	cds := CustomDimensions{
		version:       constants.Version,
		branchName:    constants.BranchName,
		userID:        userIDString,
		osName:        osName,
		osVersion:     osVersion,
		installSource: config.InstallSource(),
		machineID:     uniqID,
	}

	a := Analytics{
		client:       client,
		UniqClientID: uniqID,
		Dimensions:   &cds,
		VersInfoErr:  versInfoErr,
	}

	return &a, nil
}

func (a *Analytics) Event(category, action string) {
	a.wg.Add(1)
	go func() {
		defer a.wg.Done()
		if err := a.event(category, action); err != nil {
			logging.Debug(err.Error())
		}
	}()
}

func (a *Analytics) event(category, action string) error {
	a.client.CustomDimensionMap(a.Dimensions.toMap())

	logging.Debug("Event: %s, %s", category, action)

	if category == CatRunCmd {
		_ = a.client.sendPageview(ga.NewPageview())
	}

	return a.client.sendEvent(ga.NewEvent(category, action))
}

// EventWithLabel logs an event with a label to google analytics
func (a *Analytics) EventWithLabel(category, action, label string) {
	a.wg.Add(1)
	go func() {
		defer a.wg.Done()
		if err := a.eventWithLabel(category, action, label); err != nil {
			logging.Debug(err.Error())
		}
	}()
}

func (a *Analytics) eventWithLabel(category, action, label string) error {
	a.client.CustomDimensionMap(a.Dimensions.toMap())

	logging.Debug("Event+label: %s, %s, %s", category, action, label)

	return a.client.sendEvent(ga.NewEvent(category, action).Label(label))
}

// EventWithValue logs an event with an integer value to google analytics
func (a *Analytics) EventWithValue(category, action string, value int64) {
	a.wg.Add(1)
	go func() {
		defer a.wg.Done()
		if err := a.eventWithValue(category, action, value); err != nil {
			logging.Debug(err.Error())
		}
	}()
}

func (a *Analytics) eventWithValue(category, action string, value int64) error {
	a.client.CustomDimensionMap(a.Dimensions.toMap())

	logging.Debug("Event+value: %s, %s, %d", category, action, value)

	return a.client.sendEvent(ga.NewEvent(category, action).Value(value))
}

func (a *Analytics) SetDefer(b bool) {
	a.client.deferSend.set(b)
}

func (a *Analytics) Wait() {
	a.wg.Wait()
}

var analytics *Analytics

func init() {
	var err error
	analytics, err = NewAnalytics(
		func(vs ...interface{}) { logging.Debug(fmt.Sprint(vs...)) },
		logging.UniqID(),
		authentication.Get().UserID(),
	)
	if err != nil {
		logging.Error(err.Error())
		return
	}

	ReportMisconfig(analytics)
}

// Event logs an event to google analytics
func Event(category, action string) {
	if analytics == nil || condition.InTest() {
		return
	}
	analytics.Event(category, action)
}

// EventWithLabel logs an event with a label to google analytics
func EventWithLabel(category, action string, label string) {
	if analytics == nil || condition.InTest() {
		return
	}
	analytics.EventWithLabel(category, action, label)
}

// EventWithValue logs an event with an integer value to google analytics
func EventWithValue(category, action string, value int64) {
	if analytics == nil || condition.InTest() {
		return
	}
	analytics.EventWithValue(category, action, value)
}

func SetDimensionsOutput(output string) {
	if analytics == nil || condition.InTest() {
		return
	}
	analytics.Dimensions.SetOutput(output)
}

func ReportMisconfig(a *Analytics) {
	if a.Dimensions.osVersion == ValUnknown {
		logging.SendToRollbarWhenReady(
			"warning",
			fmt.Sprintf("Cannot detect the OS version: %v", a.VersInfoErr),
		)
	}

	if a.UniqClientID == ValUnknown {
		a.Event("error", "unknown machine id")
	}
}

func SetDefer(b bool) {
	analytics.SetDefer(b)
}

func Wait() {
	analytics.Wait()
}
