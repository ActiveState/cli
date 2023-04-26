package projectfile

import (
	"gopkg.in/yaml.v2"
)

type NameVal struct {
	Name  string `yaml:"name"`
	Value string `yaml:"value"`
}

func (nv *NameVal) UnmarshalYAML(unmarshal func(interface{}) error) error {
	data := make(map[string]interface{})
	if err := unmarshal(&data); err != nil {
		return err
	}

	if len(data) == 1 { // likely short-hand `{name}: {value}` field
		ssData, err := ensureMapStrStr(data)
		if err != nil {
			return err
		}

		for k, v := range ssData {
			nv.Name, nv.Value = k, v
			break
		}

	} else { // likely long-form `name: {name}` and `value: {value}` fields
		type Tmp NameVal
		var tmp Tmp
		if err := unmarshal(&tmp); err != nil {
			return err
		}
		*nv = NameVal(tmp)
	}

	return nil
}

// ensureMapStrStr will run the map[string]interface{} back through the yaml
// unmarshalling so that any invalid values retain the yaml package error
// messages (as much as possible).
func ensureMapStrStr(m map[string]interface{}) (map[string]string, error) {
	data, err := yaml.Marshal(m)
	if err != nil {
		return nil, err
	}

	ssm := make(map[string]string)
	if err := yaml.Unmarshal(data, &ssm); err != nil {
		return nil, err
	}

	return ssm, nil
}
