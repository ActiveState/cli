package async

import (
	"context"
	"os"
	"runtime/debug"
	"sync"
	"time"

	"github.com/ActiveState/cli/internal/analytics"
	ac "github.com/ActiveState/cli/internal/analytics/constants"
	"github.com/ActiveState/cli/internal/analytics/dimensions"
	"github.com/ActiveState/cli/internal/condition"
	"github.com/ActiveState/cli/internal/config"
	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/multilog"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/profile"
	"github.com/ActiveState/cli/internal/rtutils/ptr"
	"github.com/ActiveState/cli/internal/updater"
	"github.com/ActiveState/cli/pkg/platform/authentication"
	"github.com/ActiveState/cli/pkg/platform/model"
)

// Client is the default analytics dispatcher, forwarding analytics events to the state-svc service
type Client struct {
	svcModel         *model.SvcModel
	auth             *authentication.Auth
	output           string
	projectNameSpace string
	eventWaitGroup   *sync.WaitGroup
	sessionToken     string
	updateTag        string
	closed           bool
	sequence         int
	ci               bool
	interactive      bool
	activestateCI    bool
	source           string
}

var _ analytics.Dispatcher = &Client{}

func New(source string, svcModel *model.SvcModel, cfg *config.Instance, auth *authentication.Auth, out output.Outputer, projectNameSpace string) *Client {
	a := &Client{
		eventWaitGroup: &sync.WaitGroup{},
		source:         source,
	}

	o := string(output.PlainFormatName)
	if out.Type() != "" {
		o = string(out.Type())
	}
	a.output = o
	a.projectNameSpace = projectNameSpace
	a.auth = auth
	a.ci = condition.OnCI()
	a.interactive = out.Config().Interactive
	a.activestateCI = condition.InActiveStateCI()

	if condition.InUnitTest() {
		return a
	}

	a.svcModel = svcModel

	a.sessionToken = cfg.GetString(ac.CfgSessionToken)
	tag, ok := os.LookupEnv(constants.UpdateTagEnvVarName)
	if !ok {
		tag = cfg.GetString(updater.CfgUpdateTag)
	}
	a.updateTag = tag

	return a
}

// Event logs an event to google analytics
func (a *Client) Event(category, action string, dims ...*dimensions.Values) {
	a.EventWithLabel(category, action, "", dims...)
}

// EventWithLabel logs an event with a label to google analytics
func (a *Client) EventWithLabel(category, action, label string, dims ...*dimensions.Values) {
	a.eventWithSourceAndLabel(category, action, a.source, label, dims...)
}

// EventWithSource logs an event with another source to google analytics.
// For example, log runtime events triggered by executors as coming from an executor instead of from
// State Tool.
func (a *Client) EventWithSource(category, action, source string, dims ...*dimensions.Values) {
	a.eventWithSourceAndLabel(category, action, source, "", dims...)
}

func (a *Client) eventWithSourceAndLabel(category, action, source, label string, dims ...*dimensions.Values) {
	err := a.sendEvent(category, action, source, label, dims...)
	if err != nil {
		multilog.Error("Error during analytics.sendEvent: %v", errs.JoinMessage(err))
	}
}

// Wait can be called to ensure that all events have been processed
func (a *Client) Wait() {
	defer profile.Measure("analytics:Wait", time.Now())

	// we want Wait() to work for uninitialized Analytics
	if a == nil {
		return
	}
	a.eventWaitGroup.Wait()
}

func (a *Client) sendEvent(category, action, source, label string, dims ...*dimensions.Values) error {
	if a.svcModel == nil { // this is only true on CI
		return nil
	}

	if a.closed {
		logging.Debug("Client is closed, not sending event")
		return nil
	}

	userID := ""
	if a.auth != nil && a.auth.UserID() != nil {
		userID = string(*a.auth.UserID())
	}

	dim := dimensions.NewDefaultDimensions(a.projectNameSpace, a.sessionToken, a.updateTag, a.auth)
	dim.OutputType = &a.output
	dim.UserID = &userID
	dim.Sequence = ptr.To(a.sequence)
	a.sequence++
	dim.CI = &a.ci
	dim.Interactive = &a.interactive
	dim.ActiveStateCI = &a.activestateCI
	dim.Merge(dims...)

	dimMarshalled, err := dim.Marshal()
	if err != nil {
		return errs.Wrap(err, "Could not marshal dimensions")
	}

	a.eventWaitGroup.Add(1)
	go func() {
		defer func() { handlePanics(recover(), debug.Stack()) }()
		defer a.eventWaitGroup.Done()

		if err := a.svcModel.AnalyticsEvent(context.Background(), category, action, source, label, string(dimMarshalled)); err != nil {
			logging.Debug("Failed to report analytics event via state-svc: %s", errs.JoinMessage(err))
		}
	}()
	return nil
}

func handlePanics(err interface{}, stack []byte) {
	if err == nil {
		return
	}
	multilog.Error("Panic in client analytics: %v", err)
	logging.Debug("Stack: %s", string(stack))
}

func (a *Client) Close() {
	a.Wait()
	a.closed = true
}
