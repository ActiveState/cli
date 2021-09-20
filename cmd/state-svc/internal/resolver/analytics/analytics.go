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
	"github.com/ActiveState/cli/internal/condition"
	"github.com/ActiveState/cli/internal/config"
	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/graph"
	"github.com/ActiveState/cli/internal/installation/storage"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/machineid"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/singleton/uniqid"
	"github.com/ActiveState/cli/internal/updater"
	"github.com/ActiveState/cli/pkg/platform/authentication"
	"github.com/ActiveState/cli/pkg/platform/model"
	"github.com/ActiveState/cli/pkg/project"
	ga "github.com/ActiveState/go-ogle-analytics"
	"github.com/ActiveState/sysinfo"
)

type Resolver struct {
	gaClient         *ga.Client
	customDimensions customDimensions
	closer           func()
	ctx              context.Context
	events           chan EventData
}

type EventData struct {
	Category    string
	Action      string
	Label       string
	ProjectName string
	OutputType  string
}

type customDimensions struct {
	version       string
	branchName    string
	userID        string
	osName        string
	osVersion     string
	installSource string
	machineID     string
	uniqID        string
	sessionToken  string
	updateTag     string
}

func (d *customDimensions) toMap(projectName, projectID, output string) map[string]string {
	return map[string]string{
		// Commented out idx 1 so it's clear why we start with 2. We used to log the hostname while dogfooding internally.
		// "1": "hostname (deprected)"
		"2":  d.version,
		"3":  d.branchName,
		"4":  d.userID,
		"5":  output,
		"6":  d.osName,
		"7":  d.osVersion,
		"8":  d.installSource,
		"9":  d.machineID,
		"10": projectName,
		"11": d.sessionToken,
		"12": d.uniqID,
		"13": d.updateTag,
		"14": projectID,
	}
}

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

	var userIDString string
	userID := authentication.LegacyGet().UserID()
	if userID != nil {
		userIDString = userID.String()
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
		userID:        userIDString,
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
		events:           make(chan EventData),
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

func (r *Resolver) AnalyticsEvent(ctx context.Context, category, action string, label, projectName, out *string) (*graph.AnalyticsEventResponse, error) {
	logging.Debug("Analytics event resolver")

	lbl := ""
	if label != nil {
		lbl = *label
	}

	pn := ""
	if projectName != nil {
		pn = *projectName
	}

	o := string(output.PlainFormatName)
	if out != nil {
		o = *out
	}

	go func() {
		select {
		case r.events <- EventData{
			Category:    category,
			Action:      action,
			Label:       lbl,
			ProjectName: pn,
			OutputType:  o,
		}:
		case <-ctx.Done():
		}
	}()
	return &graph.AnalyticsEventResponse{Sent: true}, nil
}

func (r *Resolver) Close() {
	r.closer()
}

func (r *Resolver) Events() chan<- EventData {
	return r.events
}

func (r *Resolver) event(category string, action, label, projectName, output string, projectIDMap map[string]string) {
	projectID := projectID(projectIDMap, projectName)
	dimensions := r.customDimensions.toMap(projectName, projectID, output)
	r.sendGAEvent(category, action, label, dimensions)
	r.sendS3Pixel(category, action, label, dimensions)
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
	tick := time.NewTicker(time.Minute * 5)
	defer tick.Stop()

	projectIDMap := make(map[string]string)

	// flush the deferred data initially
	if err := r.flushDeferred(projectIDMap); err != nil {
		logging.Error("Failed to flush deferred data: %s", errs.JoinMessage(err))
	}
	for {
		select {
		case <-r.ctx.Done():
			return
		case <-tick.C:
			if err := r.flushDeferred(projectIDMap); err != nil {
				logging.Error("Failed to flush deferred data: %s", errs.JoinMessage(err))
			}
		case ev := <-r.events:
			r.event(ev.Category, ev.Action, ev.Label, ev.ProjectName, ev.OutputType, projectIDMap)
		}
	}
}

func (r *Resolver) flushDeferred(projectIDMap map[string]string) error {
	events, err := deferred.LoadEvents()
	if err != nil {
		return errs.Wrap(err, "Failed to load deferred events")
	}
	for _, event := range events {
		r.event(event.Category, event.Action, event.Label, event.ProjectName, event.Output, projectIDMap)
	}

	return nil
}

// projectID resolves the projectID from projectName, and caching the result in the provided projectIDMap
func projectID(projectIDMap map[string]string, projectName string) string {
	if projectName == "" {
		return ""
	}

	if pi, ok := projectIDMap[projectName]; ok {
		return pi
	}

	pn, err := project.ParseNamespace(projectName)
	if err != nil {
		logging.Error("Failed to parse project namespace %s: %s", projectName, errs.JoinMessage(err))
	}

	pj, err := model.FetchProjectByName(pn.Owner, pn.Project)
	if err != nil {
		logging.Error("Failed get project by name: %s", errs.JoinMessage(err))
	}

	pi := string(pj.ProjectID)
	projectIDMap[projectName] = pi

	return pi
}

func handlePanics(err interface{}, stack []byte) {
	if err == nil {
		return
	}
	logging.Error("Panic in client analytics: %v", err)
	logging.Debug("Stack: %s", string(stack))
}
