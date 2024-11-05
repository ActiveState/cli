package messages

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
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/graph"
	"github.com/ActiveState/cli/internal/httputil"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/poller"
	"github.com/ActiveState/cli/internal/runbits/panics"
	"github.com/ActiveState/cli/internal/strutils"
	auth "github.com/ActiveState/cli/pkg/platform/authentication"
	"github.com/ActiveState/cli/pkg/sysinfo"
	"github.com/blang/semver"
)

const ConfigKeyLastReport = "messages.last_reported"

type Messages struct {
	cfg        *config.Instance
	auth       *auth.Auth
	baseParams *ConditionParams
	poll       *poller.Poller
	checkMutex sync.Mutex
}

func New(cfg *config.Instance, auth *auth.Auth) (*Messages, error) {
	osVersion, err := sysinfo.OSVersion()
	if err != nil {
		return nil, errs.Wrap(err, "Could not get OS version")
	}

	stateVersion, err := semver.Parse(constants.Version)
	if err != nil {
		return nil, errs.Wrap(err, "Could not parse state version")
	}

	poll := poller.New(1*time.Hour, func() (interface{}, error) {
		defer func() {
			panics.LogAndPanic(recover(), debug.Stack())
		}()
		resp, err := fetch()
		return resp, err
	})

	return &Messages{
		baseParams: &ConditionParams{
			OS:           sysinfo.OS().String(),
			OSVersion:    NewVersionFromSysinfo(osVersion),
			StateChannel: constants.ChannelName,
			StateVersion: NewVersionFromSemver(stateVersion),
		},
		cfg:  cfg,
		auth: auth,
		poll: poll,
	}, nil
}

func (m *Messages) Close() error {
	m.poll.Close()
	return nil
}

func (m *Messages) Check(command string, flags []string) ([]*graph.MessageInfo, error) {
	// Prevent multiple checks at the same time, which could lead to the same message showing multiple times
	m.checkMutex.Lock()
	defer m.checkMutex.Unlock()

	cacheValue := m.poll.ValueFromCache()
	if cacheValue == nil {
		return []*graph.MessageInfo{}, nil
	}

	allMessages, ok := cacheValue.([]*graph.MessageInfo)
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

	logging.Debug("Checking %d messages with params: %#v", len(allMessages), conditionParams)

	lastReportMap := m.cfg.GetStringMap(ConfigKeyLastReport)
	msgs, err := check(&conditionParams, allMessages, lastReportMap, time.Now())
	if err != nil {
		return nil, errs.Wrap(err, "Could not check messages")
	}
	for _, msg := range msgs {
		lastReportMap[msg.ID] = time.Now().Format(time.RFC3339)
	}
	if err := m.cfg.Set(ConfigKeyLastReport, lastReportMap); err != nil {
		return nil, errs.Wrap(err, "Could not save last reported messages")
	}

	return msgs, nil
}

func check(params *ConditionParams, messages []*graph.MessageInfo, lastReportMap map[string]interface{}, baseTime time.Time) ([]*graph.MessageInfo, error) {
	funcMap := conditionFuncMap()
	filteredMessages := []*graph.MessageInfo{}
	for _, message := range messages {
		logging.Debug("Checking message %s", message.ID)
		// Ensure we don't show the same message too often
		if lastReport, ok := lastReportMap[message.ID]; ok {
			lr, ok := lastReport.(string)
			if !ok {
				return nil, errs.New("Could not get last reported time for message %s as it's not a string: %T", message.ID, lastReport)
			}
			lastReportTime, err := time.Parse(time.RFC3339, lr)
			if err != nil {
				return nil, errs.New("Could not parse last reported time for message %s as it's not a valid RFC3339 value: %v", message.ID, lastReport)
			}

			lastReportTimeAgo := baseTime.Sub(lastReportTime)
			showMessage, err := repeatValid(message.Repeat, lastReportTimeAgo)
			if err != nil {
				return nil, errs.Wrap(err, "Could not validate repeat for message %s", message.ID)
			}

			if !showMessage {
				logging.Debug("Skipping message %s as it was shown %s ago", message.ID, lastReportTimeAgo)
				continue
			}
		}

		// Validate the conditional
		if message.Condition != "" {
			result, err := strutils.ParseTemplate(fmt.Sprintf(`{{%s}}`, message.Condition), params, funcMap)
			if err != nil {
				return nil, errs.Wrap(err, "Could not parse condition template for message %s", message.ID)
			}
			if result == "true" {
				logging.Debug("Including message %s as condition %s evaluated to %s", message.ID, message.Condition, result)
				filteredMessages = append(filteredMessages, message)
			} else {
				logging.Debug("Skipping message %s as condition %s evaluated to %s", message.ID, message.Condition, result)
			}
		} else {
			logging.Debug("Including message %s as it has no condition", message.ID)
			filteredMessages = append(filteredMessages, message)
		}
	}

	return filteredMessages, nil
}

func fetch() ([]*graph.MessageInfo, error) {
	var body []byte
	var err error

	if v := os.Getenv(constants.MessagesOverrideEnvVarName); v != "" {
		body, err = fileutils.ReadFile(v)
		if err != nil {
			return nil, errs.Wrap(err, "Could not read messages override file")
		}
	} else {
		body, err = httputil.Get(constants.MessagesInfoURL)
		if err != nil {
			return nil, errs.Wrap(err, "Could not fetch messages information")
		}
	}

	var messages []*graph.MessageInfo
	if err := json.Unmarshal(body, &messages); err != nil {
		return nil, errs.Wrap(err, "Could not unmarshall messages information")
	}

	// Set defaults
	for _, message := range messages {
		if message.Placement == "" {
			message.Placement = graph.MessagePlacementTypeBeforeCmd
		}
		if message.Interrupt == "" {
			message.Interrupt = graph.MessageInterruptTypeDisabled
		}
		if message.Repeat == "" {
			message.Repeat = graph.MessageRepeatTypeDisabled
		}
	}

	return messages, nil
}

func repeatValid(repeatType graph.MessageRepeatType, lastReportTimeAgo time.Duration) (bool, error) {
	switch repeatType {
	case graph.MessageRepeatTypeConstantly:
		return true, nil
	case graph.MessageRepeatTypeDisabled:
		return false, nil
	case graph.MessageRepeatTypeHourly:
		return lastReportTimeAgo >= time.Hour, nil
	case graph.MessageRepeatTypeDaily:
		return lastReportTimeAgo >= 24*time.Hour, nil
	case graph.MessageRepeatTypeWeekly:
		return lastReportTimeAgo >= 7*24*time.Hour, nil
	case graph.MessageRepeatTypeMonthly:
		return lastReportTimeAgo >= 30*24*time.Hour, nil
	default:
		return false, errs.New("Unknown repeat type: %s", repeatType)
	}
}
