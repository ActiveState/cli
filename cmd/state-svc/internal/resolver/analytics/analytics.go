package analytics

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"runtime/debug"
	"sync"
	"time"

	"github.com/ActiveState/cli/internal/analytics"
	"github.com/ActiveState/cli/internal/analytics/deferred"
	"github.com/ActiveState/cli/internal/analytics/event"
	"github.com/ActiveState/cli/internal/condition"
	"github.com/ActiveState/cli/internal/config"
	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/graph"
	"github.com/ActiveState/cli/internal/installation/storage"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/machineid"
	"github.com/ActiveState/cli/internal/singleton/uniqid"
	"github.com/ActiveState/cli/internal/updater"
	ga "github.com/ActiveState/go-ogle-analytics"
	"github.com/ActiveState/sysinfo"
)

// AnalyticsResolver resolves requests to forward analytics events
type Resolver struct {
	gaClient         *ga.Client
	customDimensions customDimensions
	closer           func()
	ctx              context.Context
	events           chan event.EventData
}

// NewResolver initializes the resolver starting an event loop from which events are sent to the backend and initializing all the static custom dimensions
func NewResolver(cfg *config.Instance) *Resolver {
	installSource, err := storage.InstallSource()
	if err != nil {
		logging.Error("Could not detect installSource: %s", errs.Join(err, " :: ").Error())
	}

	id := machineid.UniqID()
	var trackingID string
	if !condition.InUnitTest() {
		trackingID = constants.AnalyticsTrackingID
	}

	client, err := ga.NewClient(trackingID)
	if err != nil {
		logging.Error("Cannot initialize analytics: %s", err.Error())
		client = nil
	}

	if client != nil {
		client.ClientID(id)
	}

	osName := sysinfo.OS().String()
	osVersion := "unknown"
	osvInfo, err := sysinfo.OSVersion()
	if err != nil {
		logging.Errorf("Could not detect osVersion: %v", err)
	}
	if osvInfo != nil {
		osVersion = osvInfo.Version
	}

	sessionToken := cfg.GetString(analytics.CfgSessionToken)
	tag, ok := os.LookupEnv(constants.UpdateTagEnvVarName)
	if !ok {
		tag = cfg.GetString(updater.CfgUpdateTag)
	}

	customDimensions := customDimensions{
		version:       constants.Version,
		branchName:    constants.BranchName,
		osName:        osName,
		osVersion:     osVersion,
		installSource: installSource,
		machineID:     machineid.UniqID(),
		uniqID:        uniqid.Text(),
		sessionToken:  sessionToken,
		updateTag:     tag,
	}

	if id == "unknown" {
		logging.Error("unknown machine id")
	}

	ctx, cancel := context.WithCancel(context.Background())

	r := &Resolver{
		gaClient:         client,
		customDimensions: customDimensions,
		ctx:              ctx,
		events:           make(chan event.EventData),
	}

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer handlePanics(recover(), debug.Stack())
		defer wg.Done()
		r.eventLoop()
	}()

	r.closer = func() {
		cancel()
		wg.Wait()
	}

	return r
}

// AnalyticsEvent schedules the sending of the analytics event
func (r *Resolver) AnalyticsEvent(_ context.Context, category, action string, label, projectName, out, userID *string) (*graph.AnalyticsEventResponse, error) {
	logging.Debug("Analytics event resolver")
	ev := event.New(category, action, label, projectName, out, userID)

	// We do not wait for the events to be processed, just scheduling them
	go func() {
		select {
		case r.events <- ev:
		case <-r.ctx.Done():
			// try to defer event if it cannot be scheduled in this session
			_ = deferred.DeferEvent(ev)
		}
	}()
	return &graph.AnalyticsEventResponse{Sent: true}, nil
}

// Close cancels all events, the event loop and waits for it to return
func (r *Resolver) Close() {
	r.closer()
}

// Events returns a channel to feed eventData directly to the event loop
func (r *Resolver) event(ev event.EventData) {
	dimensions := r.customDimensions.toMap(ev.ProjectName, ev.Output, ev.UserID)
	r.sendGAEvent(ev.Category, ev.Action, ev.Label, dimensions)
	r.sendS3Pixel(ev.Category, ev.Action, ev.Label, dimensions)
}

func (r *Resolver) sendGAEvent(category, action, label string, dimensions map[string]string) {
	logging.Debug("Sending Google Analytics event with: %s, %s, %s, project=%s, output=%s", category, action, label, dimensions["10"], dimensions["5"])

	r.gaClient.CustomDimensionMap(dimensions)

	if category == analytics.CatRunCmd {
		r.gaClient.Send(ga.NewPageview())
	}
	event := ga.NewEvent(category, action)
	if label != "" {
		event.Label(label)
	}
	err := r.gaClient.Send(event)
	if err != nil {
		logging.Error("Could not send GA Event: %v", err)
	}
}

func (r *Resolver) sendS3Pixel(category, action, label string, dimensions map[string]string) {
	logging.Debug("Sending S3 pixel event with: %s, %s, %s", category, action, label)
	pixelURL, err := url.Parse("https://state-tool.s3.amazonaws.com/pixel")
	if err != nil {
		logging.Error("Invalid URL for analytics S3 pixel")
		return
	}

	query := pixelURL.Query()
	query.Add("x-category", category)
	query.Add("x-action", action)
	query.Add("x-label", label)

	for num, value := range dimensions {
		key := fmt.Sprintf("x-custom%s", num)
		query.Add(key, value)
	}
	pixelURL.RawQuery = query.Encode()

	logging.Debug("Using S3 pixel URL: %v", pixelURL.String())
	_, err = http.Head(pixelURL.String())
	if err != nil {
		logging.Error("Could not download S3 pixel: %v", err)
		return
	}
}

func (r *Resolver) eventLoop() {
	// flush deferred data every five minutes
	tick := time.NewTicker(time.Minute)
	defer tick.Stop()

	// flush the event data stored to disk on start-up
	if err := r.flush(); err != nil {
		logging.Error("Failed to flush deferred data: %s", errs.JoinMessage(err))
	}
	for {
		select {
		case <-r.ctx.Done():
			return
		case <-tick.C:
			if err := r.flush(); err != nil {
				logging.Error("Failed to flush deferred data: %s", errs.JoinMessage(err))
			}
		case ev := <-r.events:
			r.event(ev)
		}
	}
}

func (r *Resolver) flush() error {
	events, err := deferred.LoadEvents()
	if err != nil {
		return errs.Wrap(err, "Failed to load deferred events")
	}
	for _, event := range events {
		r.event(event)
	}

	return nil
}

func handlePanics(err interface{}, stack []byte) {
	if err == nil {
		return
	}
	logging.Error("Panic in client analytics: %v", err)
	logging.Debug("Stack: %s", string(stack))
}
