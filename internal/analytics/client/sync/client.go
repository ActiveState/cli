package sync

import (
	"os"
	"runtime/debug"
	"strings"
	"sync"

	"github.com/ActiveState/cli/internal/analytics"
	"github.com/ActiveState/cli/internal/analytics/client/sync/reporters"
	anaConsts "github.com/ActiveState/cli/internal/analytics/constants"
	"github.com/ActiveState/cli/internal/analytics/dimensions"
	"github.com/ActiveState/cli/internal/condition"
	"github.com/ActiveState/cli/internal/config"
	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/installation/storage"
	"github.com/ActiveState/cli/internal/instanceid"
	"github.com/ActiveState/cli/internal/logging"
	configMediator "github.com/ActiveState/cli/internal/mediators/config"
	"github.com/ActiveState/cli/internal/multilog"
	"github.com/ActiveState/cli/internal/osutils"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/rollbar"
	"github.com/ActiveState/cli/internal/rtutils/ptr"
	"github.com/ActiveState/cli/internal/singleton/uniqid"
	"github.com/ActiveState/cli/internal/updater"
	"github.com/ActiveState/cli/pkg/platform/authentication"
	"github.com/ActiveState/cli/pkg/sysinfo"
)

type Reporter interface {
	ID() string
	Event(category, action, source, label string, dimensions *dimensions.Values) error
}

// Client instances send analytics events to GA and S3 endpoints without delay. It is only supposed to be used inside the `state-svc`.  All other processes should use the DefaultClient.
type Client struct {
	customDimensions *dimensions.Values
	cfg              *config.Instance
	eventWaitGroup   *sync.WaitGroup
	sendReports      bool
	reporters        []Reporter
	sequence         int
	auth             *authentication.Auth
	source           string
}

var _ analytics.Dispatcher = &Client{}

// New initializes the analytics instance with all custom dimensions known at this time
func New(source string, cfg *config.Instance, auth *authentication.Auth, out output.Outputer) *Client {
	a := &Client{
		eventWaitGroup: &sync.WaitGroup{},
		sendReports:    true,
		auth:           auth,
		source:         source,
	}

	installSource, err := storage.InstallSource()
	if err != nil {
		multilog.Error("Could not detect installSource: %s", errs.JoinMessage(err))
	}

	deviceID := uniqid.Text()

	osName := sysinfo.OS().String()
	osVersion := "unknown"
	osvInfo, err := sysinfo.OSVersion()
	if err != nil {
		multilog.Log(logging.ErrorNoStacktrace, rollbar.Error)("Could not detect osVersion: %v", err)
	}
	if osvInfo != nil {
		osVersion = osvInfo.Version
	}

	var sessionToken string
	var tag string
	if cfg != nil {
		sessionToken = cfg.GetString(anaConsts.CfgSessionToken)
		var ok bool
		tag, ok = os.LookupEnv(constants.UpdateTagEnvVarName)
		if !ok {
			tag = cfg.GetString(updater.CfgUpdateTag)
		}
		a.cfg = cfg
	}

	a.readConfig()
	configMediator.AddListener(constants.ReportAnalyticsConfig, a.readConfig)

	userID := ""
	if auth != nil && auth.UserID() != nil {
		userID = string(*auth.UserID())
	}

	interactive := false
	if out != nil {
		interactive = out.Config().Interactive
	}

	customDimensions := &dimensions.Values{
		Version:       ptr.To(constants.Version),
		ChannelName:   ptr.To(constants.ChannelName),
		OSName:        ptr.To(osName),
		OSVersion:     ptr.To(osVersion),
		InstallSource: ptr.To(installSource),
		UniqID:        ptr.To(deviceID),
		SessionToken:  ptr.To(sessionToken),
		UpdateTag:     ptr.To(tag),
		UserID:        ptr.To(userID),
		Flags:         ptr.To(dimensions.CalculateFlags()),
		InstanceID:    ptr.To(instanceid.ID()),
		Command:       ptr.To(osutils.ExecutableName()),
		Sequence:      ptr.To(0),
		CI:            ptr.To(condition.OnCI()),
		Interactive:   ptr.To(interactive),
		ActiveStateCI: ptr.To(condition.InActiveStateCI()),
	}

	a.customDimensions = customDimensions

	// Register reporters
	if condition.InTest() {
		logging.Debug("Using test reporter")
		a.NewReporter(reporters.NewTestReporter(reporters.TestReportFilepath()))
		logging.Debug("Using test reporter as instructed by env")
	} else if v := os.Getenv(constants.AnalyticsLogEnvVarName); v != "" {
		a.NewReporter(reporters.NewTestReporter(v))
	} else {
		a.NewReporter(reporters.NewPixelReporter())
	}

	return a
}

func (a *Client) readConfig() {
	doNotReport := (!a.cfg.Closed() && !a.cfg.GetBool(constants.ReportAnalyticsConfig)) ||
		strings.ToLower(os.Getenv(constants.DisableAnalyticsEnvVarName)) == "true"
	a.sendReports = !doNotReport
	logging.Debug("Sending Google Analytics reports? %v", a.sendReports)
}

func (a *Client) NewReporter(rep Reporter) {
	a.reporters = append(a.reporters, rep)
}

func (a *Client) Wait() {
	a.eventWaitGroup.Wait()
}

// Events returns a channel to feed eventData directly to the report loop
func (a *Client) report(category, action, source, label string, dimensions *dimensions.Values) {
	if !a.sendReports {
		return
	}

	for _, reporter := range a.reporters {
		if err := reporter.Event(category, action, source, label, dimensions); err != nil {
			logging.Debug(
				"Reporter failed: %s, category: %s, action: %s, error: %s",
				reporter.ID(), category, action, errs.JoinMessage(err),
			)
		}
	}
}

func (a *Client) Event(category, action string, dims ...*dimensions.Values) {
	a.EventWithLabel(category, action, "", dims...)
}

func mergeDimensions(target *dimensions.Values, dims ...*dimensions.Values) *dimensions.Values {
	actualDims := target.Clone()
	for _, dim := range dims {
		if dim == nil {
			continue
		}
		actualDims.Merge(dim)
	}
	return actualDims
}

func (a *Client) EventWithLabel(category, action, label string, dims ...*dimensions.Values) {
	a.EventWithSourceAndLabel(category, action, a.source, label, dims...)
}

// EventWithSource should only be used by clients forwarding events on behalf of another source.
// Otherwise, use Event().
func (a *Client) EventWithSource(category, action, source string, dims ...*dimensions.Values) {
	a.EventWithSourceAndLabel(category, action, source, "", dims...)
}

// EventWithSourceAndLabel should only be used by clients forwarding events on behalf of another
// source (for example, state-svc forwarding events on behalf of State Tool or an executor).
// Otherwise, use EventWithLabel().
func (a *Client) EventWithSourceAndLabel(category, action, source, label string, dims ...*dimensions.Values) {
	if a.customDimensions == nil {
		if condition.InUnitTest() {
			return
		}
		if !condition.BuiltViaCI() {
			panic("Trying to send analytics without configuring the Analytics instance.")
		}
		multilog.Critical("Trying to send analytics event without configuring the Analytics instance.")
		return
	}

	if a.auth != nil && a.auth.UserID() != nil {
		a.customDimensions.UserID = ptr.To(string(*a.auth.UserID()))
	}

	a.customDimensions.Sequence = ptr.To(a.sequence)

	actualDims := mergeDimensions(a.customDimensions, dims...)

	if a.sequence == *actualDims.Sequence {
		// Increment the sequence number unless dims overrides it (e.g. heartbeats use -1).
		a.sequence++
	}

	if err := actualDims.PreProcess(); err != nil {
		multilog.Critical("Analytics dimensions cannot be processed properly: %s", errs.JoinMessage(err))
	}

	a.eventWaitGroup.Add(1)
	// We do not wait for the events to be processed, just scheduling them
	go func() {
		defer a.eventWaitGroup.Done()
		defer func() { handlePanics(recover(), debug.Stack()) }()
		a.report(category, action, source, label, actualDims)
	}()
}

func handlePanics(err interface{}, stack []byte) {
	if err == nil {
		return
	}
	multilog.Error("Panic in state-svc analytics: %v", err)
	logging.Debug("Stack: %s", string(stack))
}

func (a *Client) Close() {
	a.Wait()
	a.sendReports = false
}
