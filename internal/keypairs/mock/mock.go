package mock

import (
	"github.com/ActiveState/cli/internal/keypairs"
)

type Mock struct {
	keypairs.Configurable
	cfg map[string]interface{}
}

func (m *Mock) ConfigPath() string {
	return ""
}

func (m *Mock) Close() error {
	return nil
}

func (m *Mock) Set(key string, value interface{}) error {
	if m.cfg == nil {
		m.cfg = make(map[string]interface{})
	}

	m.cfg[key] = value

	return nil
}

func (m *Mock) GetString(key string) string {
	if value, found := m.cfg[key]; found {
		return value.(interface{}).(string) // assume we stored the correct type
	} else {
		return ""
	}
}

func (m *Mock) GetBool(key string) bool {
	value, found := m.cfg[key]
	return found && value.(interface{}).(bool) // assume we stored the correct type
}
