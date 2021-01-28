package analytics

import (
	"encoding/json"

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

func SetDeferred(cfg Configurable, da bool) {
	deferAnalytics = da
	if deferAnalytics {
		return
	}
	eventWaitGroup.Add(1)
	go func() {
		defer eventWaitGroup.Done()
		if err := sendDeferred(cfg, sendEvent); err != nil {
			logging.Errorf("Could not send deferred events: %v", err)
		}
	}()
}

type Configurable interface {
	Set(string, interface{})
	Save() error
	GetString(string) string
}

func deferEvent(cfg Configurable, category, action, label string, dimensions map[string]string) error {
	logging.Debug("Deferring: %s, %s, %s", category, action, label)
	deferred, err := loadDeferred(cfg)
	if err != nil {
		return errs.Wrap(err, "Could not load events on defer")
	}

	deferred = append(deferred, deferredData{category, action, label, dimensions})
	if err := saveDeferred(cfg, deferred); err != nil {
		return errs.Wrap(err, "Could not save event on defer")
	}
	return nil
}

func sendDeferred(cfg Configurable, eventSender func(string, string, string, map[string]string)) error {
	deferred, err := loadDeferred(cfg)
	if err != nil {
		return errs.Wrap(err, "Could not load events on send")
	}
	for n, event := range deferred {
		eventSender(event.Category, event.Action, event.Label, event.Dimensions)
		if err := saveDeferred(cfg, deferred[n+1:]); err != nil {
			return errs.Wrap(err, "Could not save deferred event on send")
		}
	}
	if err := cfg.Save(); err != nil { // the global viper instance is bugged, need to work around it for now -- https://www.pivotaltracker.com/story/show/175624789
		return locale.WrapError(err, "err_viper_write_send_defer", "Could not save configuration on send deferred")
	}
	return nil
}

func saveDeferred(cfg Configurable, v []deferredData) error {
	s, err := json.Marshal(v)
	if err != nil {
		return errs.New("Could not serialize deferred analytics: %v, error: %v", v, err)
	}
	cfg.Set(deferredCfgKey, string(s))
	return nil
}

func loadDeferred(cfg Configurable) ([]deferredData, error) {
	v := cfg.GetString(deferredCfgKey)
	d := []deferredData{}
	if v != "" {
		err := json.Unmarshal([]byte(v), &d)
		if err != nil {
			return d, errs.Wrap(err, "Could not deserialize deferred analytics: %v", v)
		}
	}
	return d, nil
}
