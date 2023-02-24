package projectfile

type NameVal struct {
	Name  string `yaml:"name"`
	Value string `yaml:"value"`
}

func (nv *NameVal) UnmarshalYAML(unmarshal func(interface{}) error) error {
	data := make(map[string]interface{})
	if err := unmarshal(&data); err != nil {
		return err
	}

	switch entries := len(data); {
	case entries < 1:
		return nil

	case entries > 1:
		if name, ok := data["name"]; ok {
			nv.Name = name.(string)
		}
		if value, ok := data["value"]; ok {
			nv.Value = value.(string)
		}
	default:
		for k, v := range data {
			val, ok := v.(string)
			if !ok {
				continue
			}
			nv.Name, nv.Value = k, val
		}
	}

	return nil
}
