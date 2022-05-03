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
	"github.com/ActiveState/cli/internal/rtutils/p"
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
}

var _ analytics.Dispatcher = &Client{}

func New(svcModel *model.SvcModel, cfg *config.Instance, auth *authentication.Auth, out output.Outputer, projectNameSpace string) *Client {
	a := &Client{
		eventWaitGroup: &sync.WaitGroup{},
	}

	o := string(output.PlainFormatName)
	if out.Type() != "" {
		o = string(out.Type())
	}
	a.output = o
	a.projectNameSpace = projectNameSpace
	a.auth = auth

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
func (a *Client) Event(category string, action string, dims ...*dimensions.Values) {
	a.EventWithLabel(category, action, "", dims...)
}

// EventWithLabel logs an event with a label to google analytics
func (a *Client) EventWithLabel(category string, action string, label string, dims ...*dimensions.Values) {
	err := a.sendEvent(category, action, label, dims...)
	if err != nil {
		multilog.Error("Error during analytics.sendEvent: %v", errs.Join(err, ":"))
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

func (a *Client) sendEvent(category, action, label string, dims ...*dimensions.Values) error {
	if a.closed {
		logging.Debug("Client is closed, not sending event")
		return nil
	}

	userID := ""
	if a.auth != nil && a.auth.UserID() != nil {
		userID = string(*a.auth.UserID())
	}

	if a.svcModel == nil {
		if condition.InUnitTest() {
			return nil
		}
		return errs.New("Could not send analytics event, not connected to state-svc yet")
	}

	dim := dimensions.NewDefaultDimensions(a.projectNameSpace, a.sessionToken, a.updateTag)
	dim.OutputType = &a.output
	dim.UserID = &userID
	dim.Sequence = p.IntP(a.sequence)
	a.sequence++
	dim.Merge(dims...)

	dimMarshalled, err := dim.Marshal()
	if err != nil {
		return errs.Wrap(err, "Could not marshal dimensions")
	}

	a.eventWaitGroup.Add(1)
	go func() {
		defer handlePanics(recover(), debug.Stack())
		defer a.eventWaitGroup.Done()

		if err := a.svcModel.AnalyticsEvent(context.Background(), category, action, label, string(dimMarshalled)); err != nil {
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
