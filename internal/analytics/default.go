package analytics

import (
	"context"
	"runtime/debug"
	"sync"

	"github.com/ActiveState/cli/internal/analytics/deferred"
	"github.com/ActiveState/cli/internal/config"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/svcmanager"
	"github.com/ActiveState/cli/pkg/platform/model"
)

type DefaultClient struct {
	svcModel       *model.SvcModel
	output         string
	projectName    string
	isDeferred     bool
	eventWaitGroup *sync.WaitGroup
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
	if err == nil {
		return
	}
	logging.Error("Error during analytics.sendEvent: %v", errs.Join(err, ":"))
}

func (a *DefaultClient) Configure(svcMgr *svcmanager.Manager, cfg *config.Instance, out output.Outputer, projectName string) error {
	svcModel, err := model.NewSvcModel(context.Background(), cfg, svcMgr)
	if err != nil {
		return errs.Wrap(err, "Failed to initialize svc model")
	}
	o := string(output.PlainFormatName)
	if out.Type() != "" {
		o = string(out.Type())
	}
	a.svcModel = svcModel
	a.output = o
	a.projectName = projectName
	return nil
}

func (a *DefaultClient) SetDeferred(da bool) {
	a.isDeferred = da
}

func (a *DefaultClient) Wait() {
	// we want Wait() to work for uninitialized Analytics
	if a == nil {
		return
	}
	a.eventWaitGroup.Wait()
}

func (a *DefaultClient) sendEvent(category, action, label string) error {
	if a.isDeferred || a.svcModel == nil {
		if err := deferred.DeferEvent(category, action, label, a.projectName, a.output); err != nil {
			return locale.WrapError(err, "err_analytics_defer", "Could not defer event")
		}
		return nil
	}

	a.eventWaitGroup.Add(1)
	go func() {
		defer handlePanics(recover(), debug.Stack())
		defer a.eventWaitGroup.Done()
		if err := a.svcModel.AnalyticsEventWithLabel(context.Background(), category, action, label, a.projectName, a.output); err != nil {
			logging.Error("Failed to report analytics event via state-svc: %s", errs.JoinMessage(err))
		}
	}()
	return nil
}
