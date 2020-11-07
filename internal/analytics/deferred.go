package analytics

import (
	"encoding/json"

	"github.com/spf13/viper"

	"github.com/ActiveState/cli/internal/logging"
)

var Defer bool

type deferredData struct {
	Category   string
	Action     string
	Label      string
	Dimensions map[string]string
}

const deferredCfgKey = "deferrer_analytics"

func deferEvent(category, action, label string, dimensions map[string]string) error {
	logging.Debug("Deferring: %s, %s, %s", category, action, label)
	deferred := loadDeferred()
	deferred = append(deferred, deferredData{category, action, label, dimensions})
	saveDeferred(deferred)
	viper.WriteConfig() // the global viper instance is bugged, need to work around it for now -- https://www.pivotaltracker.com/story/show/175624789
	return nil
}

func sendDeferred(sender func(string, string, string, map[string]string) error) error {
	deferred := loadDeferred()
	for n, event := range deferred {
		if err := sender(event.Category, event.Action, event.Label, event.Dimensions); err != nil {
			return err
		}
		saveDeferred(deferred[n+1:])
	}
	viper.WriteConfig() // the global viper instance is bugged, need to work around it for now -- https://www.pivotaltracker.com/story/show/175624789
	return nil
}

func saveDeferred(v []deferredData) {
	s, err := json.Marshal(v)
	if err != nil {
		logging.Errorf("Could not serialize deferred analyitics: %v, error: %v", v, err)
		return
	}
	viper.Set(deferredCfgKey, string(s))
}

func loadDeferred() []deferredData {
	v := viper.GetString(deferredCfgKey)
	d := []deferredData{}
	if v != "" {
		err := json.Unmarshal([]byte(v), &d)
		if err != nil {
			logging.Errorf("Could not deserialize deferred analytics: %v, error: %v", v, err)
		}
	}
	return d
}
