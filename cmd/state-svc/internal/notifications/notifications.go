package notifications

import (
	"encoding/json"
	"fmt"
	"os"
	"runtime/debug"
	"sync"
	"time"

	"github.com/ActiveState/cli/internal/config"
	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/graph"
	"github.com/ActiveState/cli/internal/httputil"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/poller"
	"github.com/ActiveState/cli/internal/runbits/panics"
	"github.com/ActiveState/cli/internal/strutils"
	auth "github.com/ActiveState/cli/pkg/platform/authentication"
	"github.com/ActiveState/cli/pkg/sysinfo"
	"github.com/blang/semver"

	configMediator "github.com/ActiveState/cli/internal/mediators/config"
)

func init() {
	configMediator.RegisterOption(constants.NotificationsURLConfig, configMediator.String, "")
}

const ConfigKeyLastReport = "notifications.last_reported"

type Notifications struct {
	cfg        *config.Instance
	auth       *auth.Auth
	baseParams *ConditionParams
	poll       *poller.Poller
	checkMutex sync.Mutex
}

func New(cfg *config.Instance, auth *auth.Auth) (*Notifications, error) {
	osVersion, err := sysinfo.OSVersion()
	if err != nil {
		return nil, errs.Wrap(err, "Could not get OS version")
	}

	stateVersion, err := semver.Parse(constants.Version)
	if err != nil {
		return nil, errs.Wrap(err, "Could not parse state version")
	}

	poll := poller.New(10*time.Minute, func() (interface{}, error) {
		defer func() {
			panics.LogAndPanic(recover(), debug.Stack())
		}()
		resp, err := fetch(cfg)
		return resp, err
	})

	notifications := &Notifications{
		baseParams: &ConditionParams{
			OS:           sysinfo.OS().String(),
			OSVersion:    NewVersionFromSysinfo(osVersion),
			StateChannel: constants.ChannelName,
			StateVersion: NewVersionFromSemver(stateVersion),
		},
		cfg:  cfg,
		auth: auth,
		poll: poll,
	}

	configMediator.AddListener(constants.NotificationsURLConfig, func() {
		notifications.poll.Close()
		notifications.poll = poller.New(10*time.Minute, func() (interface{}, error) {
			defer func() {
				panics.LogAndPanic(recover(), debug.Stack())
			}()
			resp, err := fetch(cfg)
			return resp, err
		})
	})

	return notifications, nil
}

func (m *Notifications) Close() error {
	m.poll.Close()
	return nil
}

func (m *Notifications) Check(command string, flags []string) ([]*graph.NotificationInfo, error) {
	// Prevent multiple checks at the same time, which could lead to the same notification showing multiple times
	m.checkMutex.Lock()
	defer m.checkMutex.Unlock()

	cacheValue := m.poll.ValueFromCache()
	if cacheValue == nil {
		return []*graph.NotificationInfo{}, nil
	}

	allNotifications, ok := cacheValue.([]*graph.NotificationInfo)
	if !ok {
		return nil, errs.New("cacheValue has unexpected type: %T", cacheValue)
	}

	conditionParams := *m.baseParams // copy
	conditionParams.UserEmail = m.auth.Email()
	conditionParams.UserName = m.auth.WhoAmI()
	conditionParams.Command = command
	conditionParams.Flags = flags

	if id := m.auth.UserID(); id != nil {
		conditionParams.UserID = id.String()
	}

	logging.Debug("Checking %d notifications with params: %#v", len(allNotifications), conditionParams)

	lastReportMap := m.cfg.GetStringMap(ConfigKeyLastReport)
	msgs, err := check(&conditionParams, allNotifications, lastReportMap, time.Now())
	if err != nil {
		return nil, errs.Wrap(err, "Could not check notifications")
	}
	for _, msg := range msgs {
		lastReportMap[msg.ID] = time.Now().Format(time.RFC3339)
	}
	if err := m.cfg.Set(ConfigKeyLastReport, lastReportMap); err != nil {
		return nil, errs.Wrap(err, "Could not save last reported notifications")
	}

	return msgs, nil
}

func notificationInDateRange(notification *graph.NotificationInfo, baseTime time.Time) (bool, error) {
	if notification.StartDate != "" {
		startDate, err := time.Parse(time.RFC3339, notification.StartDate)
		if err != nil {
			return false, errs.Wrap(err, "Could not parse start date for notification %s", notification.ID)
		}
		if baseTime.Before(startDate) {
			return false, nil
		}
	}

	if notification.EndDate != "" {
		endDate, err := time.Parse(time.RFC3339, notification.EndDate)
		if err != nil {
			return false, errs.Wrap(err, "Could not parse end date for notification %s", notification.ID)
		}
		if baseTime.After(endDate) {
			return false, nil
		}
	}

	return true, nil
}

func check(params *ConditionParams, notifications []*graph.NotificationInfo, lastReportMap map[string]interface{}, baseTime time.Time) ([]*graph.NotificationInfo, error) {
	funcMap := conditionFuncMap()
	filteredNotifications := []*graph.NotificationInfo{}
	for _, notification := range notifications {
		logging.Debug("Checking notification %s", notification.ID)

		// Ensure we don't show the same message too often
		if lastReport, ok := lastReportMap[notification.ID]; ok {
			lr, ok := lastReport.(string)
			if !ok {
				return nil, errs.New("Could not get last reported time for notification %s as it's not a string: %T", notification.ID, lastReport)
			}
			lastReportTime, err := time.Parse(time.RFC3339, lr)
			if err != nil {
				return nil, errs.New("Could not parse last reported time for notification %s as it's not a valid RFC3339 value: %v", notification.ID, lastReport)
			}

			lastReportTimeAgo := baseTime.Sub(lastReportTime)
			showNotification, err := repeatValid(notification.Repeat, lastReportTimeAgo)
			if err != nil {
				return nil, errs.Wrap(err, "Could not validate repeat for notification %s", notification.ID)
			}

			if !showNotification {
				logging.Debug("Skipping notification %s as it was shown %s ago", notification.ID, lastReportTimeAgo)
				continue
			}
		}

		// Check if message is within date range
		inRange, err := notificationInDateRange(notification, baseTime)
		if err != nil {
			return nil, errs.Wrap(err, "Could not check if notification %s is in date range", notification.ID)
		}
		if !inRange {
			logging.Debug("Skipping notification %s as it is outside of its date range", notification.ID)
			continue
		}

		// Validate the conditional
		if notification.Condition != "" {
			result, err := strutils.ParseTemplate(fmt.Sprintf(`{{%s}}`, notification.Condition), params, funcMap)
			if err != nil {
				logging.Warning("Could not parse condition template for notification %s: %v", notification.ID, err)
				continue
			}
			if result == "true" {
				logging.Debug("Including notification %s as condition %s evaluated to %s", notification.ID, notification.Condition, result)
				filteredNotifications = append(filteredNotifications, notification)
			} else {
				logging.Debug("Skipping notification %s as condition %s evaluated to %s", notification.ID, notification.Condition, result)
			}
		} else {
			logging.Debug("Including notification %s as it has no condition", notification.ID)
			filteredNotifications = append(filteredNotifications, notification)
		}
	}

	return filteredNotifications, nil
}

func fetch(cfg *config.Instance) ([]*graph.NotificationInfo, error) {
	var body []byte
	var err error

	var (
		notificationsURL string

		envURL    = os.Getenv(constants.NotificationsOverrideEnvVarName)
		configURL = cfg.GetString(constants.NotificationsURLConfig)
	)

	switch {
	case envURL != "":
		notificationsURL = envURL
	case configURL != "":
		notificationsURL = configURL
	default:
		notificationsURL = constants.NotificationsInfoURL
	}

	logging.Debug("Fetching notifications from %s", notificationsURL)
	// Check if this is a local file path (when using environment override)
	if envURL != "" {
		body, err = os.ReadFile(notificationsURL)
		if err != nil {
			return nil, errs.Wrap(err, "Could not read notifications file")
		}
	} else {
		// Use HTTP client for remote URLs
		body, err = httputil.Get(notificationsURL)
		if err != nil {
			return nil, errs.Wrap(err, "Could not fetch notifications information")
		}
	}

	var notifications []*graph.NotificationInfo
	if err := json.Unmarshal(body, &notifications); err != nil {
		return nil, errs.Wrap(err, "Could not unmarshall notifications information")
	}

	// Set defaults
	for _, notification := range notifications {
		if notification.Placement == "" {
			notification.Placement = graph.NotificationPlacementTypeBeforeCmd
		}
		if notification.Interrupt == "" {
			notification.Interrupt = graph.NotificationInterruptTypeDisabled
		}
		if notification.Repeat == "" {
			notification.Repeat = graph.NotificationRepeatTypeDisabled
		}
	}

	return notifications, nil
}

func repeatValid(repeatType graph.NotificationRepeatType, lastReportTimeAgo time.Duration) (bool, error) {
	switch repeatType {
	case graph.NotificationRepeatTypeConstantly:
		return true, nil
	case graph.NotificationRepeatTypeDisabled:
		return false, nil
	case graph.NotificationRepeatTypeHourly:
		return lastReportTimeAgo >= time.Hour, nil
	case graph.NotificationRepeatTypeDaily:
		return lastReportTimeAgo >= 24*time.Hour, nil
	case graph.NotificationRepeatTypeWeekly:
		return lastReportTimeAgo >= 7*24*time.Hour, nil
	case graph.NotificationRepeatTypeMonthly:
		return lastReportTimeAgo >= 30*24*time.Hour, nil
	default:
		return false, errs.New("Unknown repeat type: %s", repeatType)
	}
}
