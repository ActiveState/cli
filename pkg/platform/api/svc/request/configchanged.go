package request

type ConfigChanged struct {
	key string
}

func NewConfigChanged(key string) *ConfigChanged {
	return &ConfigChanged{key: key}
}

func (e *ConfigChanged) Query() string {
	return `query($key: String!) {
	    configChanged(key: $key) {
	      received
	    }
	}`
}

func (e *ConfigChanged) Vars() (map[string]interface{}, error) {
	return map[string]interface{}{"key": e.key}, nil
}
