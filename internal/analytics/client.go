package analytics

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"runtime/debug"
	"sync"
	"time"

	"github.com/ActiveState/cli/internal/analytics/dimensions"
	ac "github.com/ActiveState/cli/internal/analytics/constants"
	"github.com/ActiveState/cli/internal/appinfo"
	"github.com/ActiveState/cli/internal/condition"
	"github.com/ActiveState/cli/internal/config"
	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/exeutils"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/profile"
	"github.com/ActiveState/cli/internal/svcmanager"
	"github.com/ActiveState/cli/internal/updater"
	"github.com/ActiveState/cli/pkg/platform/authentication"
	"github.com/ActiveState/cli/pkg/platform/model"
)

// DefaultClient is the default analytics dispatcher, forwarding analytics events to the state-svc service
type DefaultClient struct {
	svcModel         *model.SvcModel
	auth             *authentication.Auth
	output           string
	projectNameSpace string
	eventWaitGroup   *sync.WaitGroup

	sessionToken string
	updateTag string
}

func New() *DefaultClient {
	return &DefaultClient{
		eventWaitGroup: &sync.WaitGroup{},
	}
}

// Event logs an event to google analytics
func (a *DefaultClient) Event(category string, action string) {
	a.EventWithLabel(category, action, "")
}

// EventWithLabel logs an event with a label to google analytics
func (a *DefaultClient) EventWithLabel(category string, action string, label string) {
	err := a.sendEvent(category, action, label)
	if err != nil {
		logging.Error("Error during analytics.sendEvent: %v", errs.Join(err, ":"))
	}
}

// Configure configures the default client, connecting it to a state-svc service
func (a *DefaultClient) Configure(svcMgr *svcmanager.Manager, cfg *config.Instance, auth *authentication.Auth, out output.Outputer, projectNameSpace string) error {
	o := string(output.PlainFormatName)
	if out.Type() != "" {
		o = string(out.Type())
	}
	a.output = o
	a.projectNameSpace = projectNameSpace
	a.auth = auth

	if condition.InUnitTest() {
		return nil
	}

	svcModel, err := model.NewSvcModel(context.Background(), cfg, svcMgr)
	if err != nil {
		return errs.Wrap(err, "Failed to initialize svc model")
	}
	a.svcModel = svcModel

	a.sessionToken = cfg.GetString(ac.CfgSessionToken)
	tag, ok := os.LookupEnv(constants.UpdateTagEnvVarName)
	if !ok {
		tag = cfg.GetString(updater.CfgUpdateTag)
	}
	a.updateTag = tag

	return nil
}

// Wait can be called to ensure that all events have been processed
func (a *DefaultClient) Wait() {
	defer profile.Measure("analytics:Wait", time.Now())

	// we want Wait() to work for uninitialized Analytics
	if a == nil {
		return
	}
	a.eventWaitGroup.Wait()
}

func (a *DefaultClient) sendEvent(category, action, label string) error {
	userID := ""
	if a.auth != nil && a.auth.UserID() != nil {
		userID = string(*a.auth.UserID())
	}

	a.eventWaitGroup.Add(1)
	go func() {
		a.sendS3Pixel(category, action, label)
		a.eventWaitGroup.Done()
	}()

	if a.svcModel == nil {
		if condition.InUnitTest() {
			return nil
		}
		return errs.New("Could not send analytics event, not connected to state-svc yet")
	}

	a.eventWaitGroup.Add(1)
	go func() {
		defer handlePanics(recover(), debug.Stack())
		defer a.eventWaitGroup.Done()
		if err := a.svcModel.AnalyticsEventWithLabel(context.Background(), category, action, label, a.projectNameSpace, a.output, userID); err != nil {
			logging.Error("Failed to report analytics event via state-svc: %s", errs.JoinMessage(err))
		}
	}()
	return nil
}

func (a *DefaultClient) sendS3Pixel(category, action, label string) {
	defer handlePanics(recover(), debug.Stack())
	defer profile.Measure("sendS3Pixel", time.Now())

	query := &url.Values{}
	query.Add("x-category", category)
	query.Add("x-action", action)
	query.Add("x-label", label)

	for num, value := range dimensions.NewDefaultDimensions(a.projectNameSpace, a.sessionToken, a.updateTag).ToMap() {
		key := fmt.Sprintf("x-custom%s", num)
		query.Add(key, value)
	}
	fullQuery := query.Encode()

	logging.Debug("Using S3 pixel query: %v", fullQuery)
	svcExec := appinfo.SvcApp().Exec()
	exeutils.ExecuteAndForget(svcExec, []string{"_event", query.Encode()})
}

func handlePanics(err interface{}, stack []byte) {
	if err == nil {
		return
	}
	logging.Error("Panic in client analytics: %v", err)
	logging.Debug("Stack: %s", string(stack))
}
