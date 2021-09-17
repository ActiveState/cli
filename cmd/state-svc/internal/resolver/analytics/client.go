package analytics

import (
	"runtime/debug"
	"sync"

	"github.com/ActiveState/cli/internal/analytics/deferred"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
)

type Client struct {
	events         chan<- EventData
	eventWaitGroup *sync.WaitGroup
	isDeferred     bool
}

func NewClient() *Client {
	return &Client{eventWaitGroup: &sync.WaitGroup{}}
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

func (c *Client) Configure(events chan<- EventData) {
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
	// TODO: Once, we execute project-specific tasks in the state-svc that can be bound to a specific State Tool instance, we need to add a function that also encapsulates this information
	projectName := ""
	outputType := ""

	// if events channel is not set yet, we will defer the events to the file system
	if c.isDeferred || c.events == nil {
		if err := deferred.DeferEvent(category, action, label, projectName, outputType); err != nil {
			return locale.WrapError(err, "err_analytics_defer", "Could not defer event")
		}
		return nil
	}

	c.eventWaitGroup.Add(1)
	go func() {
		defer handlePanics(recover(), debug.Stack())
		defer c.eventWaitGroup.Done()
		c.events <- EventData{category, action, label, projectName, outputType}
	}()
	return nil
}

// TODO: Implement this function that returns a copy of this client tied to a project (and maybe a State Tool outputType if applicable)
// func (c *Client) BindToProject(projectName, outputType string) *Client { }
