package projectfile

import (
	"bytes"

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

	if len(data) == 1 {
		ssData, err := sanitizeMap(data)
		if err != nil {
			return err
		}

		for k, v := range ssData {
			nv.Name, nv.Value = k, v
			break
		}

	} else {
		for k := range data {
			if k != "name" && k != "value" {
				delete(data, k)
			}
		}

		ssData, err := sanitizeMap(data)
		if err != nil {
			return err
		}

		nv.Name, nv.Value = ssData["name"], ssData["value"]
	}

	return nil
}

func sanitizeMap(m map[string]interface{}) (map[string]string, error) {
	data, err := yaml.Marshal(m)
	if err != nil {
		return nil, err
	}

	data = bytes.TrimSpace(data)

	ssm := make(map[string]string)
	if err := yaml.Unmarshal(data, &ssm); err != nil {
		return nil, err
	}

	return ssm, nil
}
