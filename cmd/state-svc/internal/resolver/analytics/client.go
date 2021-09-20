package analytics

import (
	"runtime/debug"
	"sync"

	"github.com/ActiveState/cli/internal/analytics/deferred"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/pkg/platform/authentication"
)

// Client is an AnalyticsDispatcher that is supposed to be use by the `state-svc` service only. It sends the events directly to the analytics event loop started in the resolver (once configured)
type Client struct {
	auth           *authentication.Auth
	events         chan<- deferred.EventData
	eventWaitGroup *sync.WaitGroup
	isDeferred     bool
}

func NewClient() *Client {
	return &Client{auth: authentication.LegacyGet(), eventWaitGroup: &sync.WaitGroup{}}
}

func (c *Client) Event(category string, action string) {
	c.EventWithLabel(category, action, "")
}

func (c *Client) EventWithLabel(category string, action, label string) {
	err := c.sendEvent(category, action, label)
	if err == nil {
		return
	}
	logging.Error("Error during analytics.sendEvent: %v", errs.Join(err, ":"))
}

// Configure ties this client to the events loop running as part of the Resolver, un-configured clients defer events to the hard-drive
func (c *Client) Configure(events chan<- deferred.EventData) {
	c.events = events
}

func (c *Client) SetDeferred(da bool) {
	c.isDeferred = true
}

func (c *Client) Wait() {
	c.eventWaitGroup.Wait()
}

func (c *Client) sendEvent(category, action, label string) error {
	// For now analytics events triggered by the state-svc are NEVER bound to a project or an output-type
	projectName := ""
	outputType := ""
	userID := ""
	if c.auth != nil && c.auth.UserID() != nil {
		userID = string(*c.auth.UserID())
	}

	// if events channel is not set yet, we will defer the events to the file system
	if c.isDeferred || c.events == nil {
		if err := deferred.DeferEvent(category, action, label, projectName, outputType, userID); err != nil {
			return locale.WrapError(err, "err_analytics_defer", "Could not defer event")
		}
		return nil
	}

	// we do not wait for the event to be processed, it is just scheduled in the background
	c.eventWaitGroup.Add(1)
	go func() {
		defer handlePanics(recover(), debug.Stack())
		defer c.eventWaitGroup.Done()
		c.events <- deferred.EventData{category, action, label, projectName, outputType, userID}
	}()
	return nil
}

// TODO: Implement this function that returns a copy of this client tied to a project (and maybe a State Tool outputType if applicable)
// func (c *Client) BindToProject(projectName, outputType string) *Client { }
