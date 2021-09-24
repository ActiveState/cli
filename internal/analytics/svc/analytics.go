package svc

import (
	"fmt"
	"net/http"
	"net/url"
	"os"
	"runtime/debug"
	"sync"
	"time"

	anaConsts "github.com/ActiveState/cli/internal/analytics/constants"
	"github.com/ActiveState/cli/internal/condition"
	"github.com/ActiveState/cli/internal/config"
	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/installation/storage"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/machineid"
	"github.com/ActiveState/cli/internal/singleton/uniqid"
	"github.com/ActiveState/cli/internal/updater"
	"github.com/ActiveState/cli/pkg/platform/authentication"
	"github.com/ActiveState/cli/pkg/platform/model"
	"github.com/ActiveState/cli/pkg/project"
	ga "github.com/ActiveState/go-ogle-analytics"
	"github.com/ActiveState/sysinfo"
	"github.com/patrickmn/go-cache"
)

// Analytics instances send analytics events to GA and S3 endpoints without delay. It is only supposed to be used inside the `state-svc`.  All other processes should use the DefaultClient.
type Analytics struct {
	gaClient         *ga.Client
	customDimensions *CustomDimensions
	eventWaitGroup   *sync.WaitGroup
	projectIDCache   *cache.Cache
	projectIDMutex   *sync.Mutex // used to synchronize API calls resolving the projectID
}

// New initializes the analytics instance with all custom dimensions known at this time
func New() *Analytics {
	r := &Analytics{
		eventWaitGroup: &sync.WaitGroup{},
		projectIDCache: cache.New(30*time.Minute, time.Hour),
		projectIDMutex: &sync.Mutex{},
	}

	return r
}

func (a *Analytics) Configure(cfg *config.Instance, auth *authentication.Auth) {
	installSource, err := storage.InstallSource()
	if err != nil {
		logging.Error("Could not detect installSource: %s", errs.Join(err, " :: ").Error())
	}

	id := machineid.UniqID()
	if id == "unknown" {
		logging.Error("unknown machine id")
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

	sessionToken := cfg.GetString(anaConsts.CfgSessionToken)
	tag, ok := os.LookupEnv(constants.UpdateTagEnvVarName)
	if !ok {
		tag = cfg.GetString(updater.CfgUpdateTag)
	}

	// TODO At some point we want to refresh this whenever the authentication changes https://www.pivotaltracker.com/story/show/179703938
	userID := ""
	if auth != nil && auth.UserID() != nil {
		userID = string(*auth.UserID())
	}

	customDimensions := &CustomDimensions{
		version:       constants.Version,
		branchName:    constants.BranchName,
		osName:        osName,
		osVersion:     osVersion,
		installSource: installSource,
		machineID:     machineid.UniqID(),
		uniqID:        uniqid.Text(),
		sessionToken:  sessionToken,
		updateTag:     tag,
		userID:        userID,
	}

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

	a.customDimensions = customDimensions
	a.gaClient = client
}

func (a *Analytics) DimensionsWithClientData(projectNameSpace, outputType, userID string) *CustomDimensions {
	return a.customDimensions.WithClientData(projectNameSpace, outputType, userID)
}

// SendWithCustomDimensions sends an analytics event with the given custom dimensions
func (a *Analytics) SendWithCustomDimensions(category, action, label string, dims *CustomDimensions) {
	if a.customDimensions == nil {
		if !condition.BuiltViaCI() {
			panic("Trying to send analytics without configuring the Analytics instance.")
		}
		logging.Critical("Trying to send analytics event without configuring the Analytics instance.")
		return
	}
	if dims.uniqID == machineid.FallbackID {
		logging.Critical("machine id was set to fallback id when creating analytics event")
	}

	logging.Debug("Analytics event resolver")

	a.eventWaitGroup.Add(1)
	// We do not wait for the events to be processed, just scheduling them
	go func() {
		defer a.eventWaitGroup.Done()
		defer handlePanics(recover(), debug.Stack())
		dims.projectID = a.projectID(dims.projectName)
		a.event(category, action, label, dims)
	}()
}

func (a *Analytics) Wait() {
	a.eventWaitGroup.Wait()
}

// Events returns a channel to feed eventData directly to the event loop
func (a *Analytics) event(category, action, label string, dimensions *CustomDimensions) {
	dims := dimensions.toMap()
	a.sendGAEvent(category, action, label, dims)
	a.sendS3Pixel(category, action, label, dims)
}

func (a *Analytics) sendGAEvent(category, action, label string, dimensions map[string]string) {
	logging.Debug("Sending Google Analytics event with: %s, %s, %s, project=%s, output=%s", category, action, label, dimensions["10"], dimensions["5"])

	a.gaClient.CustomDimensionMap(dimensions)

	if category == anaConsts.CatRunCmd {
		a.gaClient.Send(ga.NewPageview())
	}
	event := ga.NewEvent(category, action)
	if label != "" {
		event.Label(label)
	}
	err := a.gaClient.Send(event)
	if err != nil {
		logging.Error("Could not send GA Event: %v", err)
	}
}

func (a *Analytics) sendS3Pixel(category, action, label string, dimensions map[string]string) {
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
	}
}

func (a *Analytics) Event(category string, action string) {
	a.EventWithLabel(category, action, "")
}

func (a *Analytics) EventWithLabel(category string, action, label string) {
	a.SendWithCustomDimensions(category, action, label, a.customDimensions)
}

// projectID resolves the projectID from projectName and caches the result in the provided projectIDMap
func (r *Analytics) projectID(projectName string) string {
	if projectName == "" {
		return ""
	}

	// Lock mutex to prevent resolving the same projectName more than once
	r.projectIDMutex.Lock()
	defer r.projectIDMutex.Unlock()

	if pi, ok := r.projectIDCache.Get(projectName); ok {
		return pi.(string)
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
	r.projectIDCache.Set(projectName, pi, cache.DefaultExpiration)

	return pi
}

func handlePanics(err interface{}, stack []byte) {
	if err == nil {
		return
	}
	logging.Error("Panic in state-svc analytics: %v", err)
	logging.Debug("Stack: %s", string(stack))
}
