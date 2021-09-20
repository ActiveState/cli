package mock

import (
	"github.com/ActiveState/cli/internal/config"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/svcmanager"
	"github.com/ActiveState/cli/pkg/platform/authentication"
)

type EventData struct {
	Category string
	Action   string
	Label    string
}

type Mock struct {
	Events          []EventData
	IsDeferred      bool
	CalledWait      bool
	CalledConfigure bool
}

func New() *Mock {
	return &Mock{}
}

func (m *Mock) Event(category string, action string) {
	m.EventWithLabel(category, action, "")
}

func (m *Mock) EventWithLabel(category string, action string, label string) {
	m.Events = append(m.Events, EventData{category, action, label})
}

func (m *Mock) SetDeferred(da bool) {
	m.IsDeferred = da
}

func (m *Mock) Wait() {
	m.CalledWait = true
}

func (m *Mock) Configure(svcMgr *svcmanager.Manager, cfg *config.Instance, auth *authentication.Auth, out output.Outputer, projectName string) error {
	m.CalledConfigure = true
	return nil
}
