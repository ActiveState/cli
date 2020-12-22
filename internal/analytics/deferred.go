package analytics

import (
	"encoding/json"

	"github.com/ActiveState/cli/internal/config"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
)

var deferAnalytics bool

type deferredData struct {
	Category   string
	Action     string
	Label      string
	Dimensions map[string]string
}

const deferredCfgKey = "deferrer_analytics"

func SetDeferred(da bool) {
	deferAnalytics = da
	if deferAnalytics {
		return
	}
	eventWaitGroup.Add(1)
	go func() {
		defer eventWaitGroup.Done()
		if err := sendDeferred(sendEvent); err != nil {
			logging.Errorf("Could not send deferred events: %v", err)
		}
	}()
}

func deferEvent(category, action, label string, dimensions map[string]string) error {
	logging.Debug("Deferring: %s, %s, %s", category, action, label)
	deferred, err := loadDeferred()
	if err != nil {
		return errs.Wrap(err, "Could not load events on defer")
	}

	deferred = append(deferred, deferredData{category, action, label, dimensions})
	if err := saveDeferred(deferred); err != nil {
		return errs.Wrap(err, "Could not save event on defer")
	}
	return nil
}

func sendDeferred(sender func(string, string, string, map[string]string) error) error {
	deferred, err := loadDeferred()
	if err != nil {
		return errs.Wrap(err, "Could not load events on send")
	}
	for n, event := range deferred {
		if err := sender(event.Category, event.Action, event.Label, event.Dimensions); err != nil {
			return errs.Wrap(err, "Could not send deferred event")
		}
		if err := saveDeferred(deferred[n+1:]); err != nil {
			return errs.Wrap(err, "Could not save deferred event on send")
		}
	}
	if err := config.Get().WriteConfig(); err != nil { // the global viper instance is bugged, need to work around it for now -- https://www.pivotaltracker.com/story/show/175624789
		return locale.WrapError(err, "err_viper_write_send_defer", "Could not save configuration on send deferred")
	}
	return nil
}

func saveDeferred(v []deferredData) error {
	s, err := json.Marshal(v)
	if err != nil {
		return errs.New("Could not serialize deferred analytics: %v, error: %v", v, err)
	}
	config.Get().Set(deferredCfgKey, string(s))
	return nil
}

func loadDeferred() ([]deferredData, error) {
	v := config.Get().GetString(deferredCfgKey)
	d := []deferredData{}
	if v != "" {
		err := json.Unmarshal([]byte(v), &d)
		if err != nil {
			return d, errs.Wrap(err, "Could not deserialize deferred analytics: %v", v)
		}
	}
	return d, nil
}
