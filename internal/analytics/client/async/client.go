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
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/profile"
	"github.com/ActiveState/cli/internal/rtutils/p"
	"github.com/ActiveState/cli/internal/svcmanager"
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

	sessionToken string
	updateTag    string
}

var _ analytics.Dispatcher = &Client{}

func New(svcMgr *svcmanager.Manager, cfg *config.Instance, auth *authentication.Auth, out output.Outputer, projectNameSpace string) *Client {
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

	a.svcModel = model.NewSvcModel(cfg, svcMgr)

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
		logging.Error("Error during analytics.sendEvent: %v", errs.Join(err, ":"))
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
	userID := ""
	if a.auth != nil && a.auth.UserID() != nil {
		userID = string(*a.auth.UserID())
	}

	a.eventWaitGroup.Add(1)
	go func() {
		actualDims := dimensions.NewDefaultDimensions(a.projectNameSpace, a.sessionToken, a.updateTag)
		for _, dim := range dims {
			actualDims.Merge(dim)
		}

		a.eventWaitGroup.Done()
	}()

	if a.svcModel == nil {
		if condition.InUnitTest() {
			return nil
		}
		return errs.New("Could not send analytics event, not connected to state-svc yet")
	}

	dim := &dimensions.Values{
		ProjectNameSpace: p.StrP(a.projectNameSpace),
		OutputType:       p.StrP(a.output),
		UserID:           p.StrP(userID),
	}
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
			logging.Error("Failed to report analytics event via state-svc: %s", errs.JoinMessage(err))
		}
	}()
	return nil
}

func handlePanics(err interface{}, stack []byte) {
	if err == nil {
		return
	}
	logging.Error("Panic in client analytics: %v", err)
	logging.Debug("Stack: %s", string(stack))
}
