package sync

import (
	"os"
	"runtime/debug"
	"strings"
	"sync"

	internalAnalytics "github.com/ActiveState/cli/internal/analytics"
	anaConsts "github.com/ActiveState/cli/internal/analytics/constants"
	"github.com/ActiveState/cli/internal/condition"
	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/installation/storage"
	"github.com/ActiveState/cli/internal/instanceid"
	"github.com/ActiveState/cli/internal/logging"
	configMediator "github.com/ActiveState/cli/internal/mediators/config"
	"github.com/ActiveState/cli/internal/multilog"
	"github.com/ActiveState/cli/internal/osutils"
	"github.com/ActiveState/cli/internal/rollbar"
	"github.com/ActiveState/cli/internal/rtutils/p"
	"github.com/ActiveState/cli/internal/singleton/uniqid"
	"github.com/ActiveState/cli/internal/updater"
	"github.com/ActiveState/cli/pkg/platform/analytics"
	"github.com/ActiveState/cli/pkg/platform/analytics/sync/reporters"
	"github.com/ActiveState/cli/pkg/sysinfo"
	"github.com/go-openapi/strfmt"
)

type Reporter interface {
	ID() string
	Event(category, action, label string, dimensions *analytics.Dimensions) error
}

// Client instances send analytics events to GA and S3 endpoints without delay. It is only supposed to be used inside the `state-svc`.  All other processes should use the DefaultClient.
type Client struct {
	customDimensions *analytics.Dimensions
	eventWaitGroup   *sync.WaitGroup
	sendReports      bool
	reporters        []Reporter
	sequence         int
	cfg              Configer
	auth             Auther
}

type Configer interface {
	GetString(string) string
	GetBool(string) bool
	IsSet(string) bool
	Closed() bool
}

type Auther interface {
	UserID() *strfmt.UUID
}

var _ internalAnalytics.Dispatcher = &Client{}

// New initializes the analytics instance with all custom dimensions known at this time
func New(cfg Configer, auth Auther, version, branchName string) *Client {
	a := &Client{
		eventWaitGroup: &sync.WaitGroup{},
		sendReports:    true,
		auth:           auth,
	}

	installSource, err := storage.InstallSource()
	if err != nil {
		multilog.Error("Could not detect installSource: %s", errs.Join(err, " :: ").Error())
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

	customDimensions := &analytics.Dimensions{
		Version:       p.StrP(version),
		BranchName:    p.StrP(branchName),
		OSName:        p.StrP(osName),
		OSVersion:     p.StrP(osVersion),
		InstallSource: p.StrP(installSource),
		UniqID:        p.StrP(deviceID),
		SessionToken:  p.StrP(sessionToken),
		UpdateTag:     p.StrP(tag),
		UserID:        p.StrP(userID),
		Flags:         p.StrP(internalAnalytics.CalculateFlags()),
		InstanceID:    p.StrP(instanceid.ID()),
		Command:       p.StrP(osutils.ExecutableName()),
		Sequence:      p.IntP(0),
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
	doNotReport := (!a.cfg.Closed() && a.cfg.IsSet(constants.ReportAnalyticsConfig) && !a.cfg.GetBool(constants.ReportAnalyticsConfig)) ||
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
func (a *Client) report(category, action, label string, dimensions *analytics.Dimensions) {
	if !a.sendReports {
		return
	}

	for _, reporter := range a.reporters {
		if err := reporter.Event(category, action, label, dimensions); err != nil {
			logging.Debug(
				"Reporter failed: %s, category: %s, action: %s, error: %s",
				reporter.ID(), category, action, errs.JoinMessage(err),
			)
		}
	}
}

func (a *Client) Event(category string, action string, dims ...*analytics.Dimensions) {
	a.EventWithLabel(category, action, "", dims...)
}

func mergeDimensions(target *analytics.Dimensions, dims ...*analytics.Dimensions) *analytics.Dimensions {
	actualDims := target.Clone()
	for _, dim := range dims {
		if dim == nil {
			continue
		}
		actualDims.Merge(dim)
	}
	return actualDims
}

func (a *Client) EventWithLabel(category string, action, label string, dims ...*analytics.Dimensions) {
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
		a.customDimensions.UserID = p.StrP(string(*a.auth.UserID()))
	}

	a.customDimensions.Sequence = p.IntP(a.sequence)
	a.sequence++

	actualDims := mergeDimensions(a.customDimensions, dims...)

	if err := actualDims.PreProcess(); err != nil {
		multilog.Critical("Analytics dimensions cannot be processed properly: %s", errs.JoinMessage(err))
	}

	a.eventWaitGroup.Add(1)
	// We do not wait for the events to be processed, just scheduling them
	go func() {
		defer a.eventWaitGroup.Done()
		defer func() { handlePanics(recover(), debug.Stack()) }()
		a.report(category, action, label, actualDims)
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
