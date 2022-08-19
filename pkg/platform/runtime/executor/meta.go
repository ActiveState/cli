package executor

import (
	"github.com/ActiveState/cli/internal/svcctl"
	"github.com/ActiveState/cli/pkg/project"
)

type Meta struct {
	SockPath   string
	Env        map[string]string
	CommitUUID string
	Namespace  string
	Headless   bool
}

func NewMeta(env map[string]string, t Targeter) *Meta {
	commitID := t.CommitUUID().String()
	return &Meta{
		SockPath:   svcctl.NewIPCSockPathFromGlobals().String(),
		Env:        env,
		CommitUUID: commitID,
		Namespace:  project.NewNamespace(t.Owner(), t.Name(), commitID).String(),
		Headless:   t.Headless(),
	}
}

func NewMetaFromDisk(dir string) (*Meta, error) {
	return nil, nil
}

func (m *Meta) WriteToDisk(dir string) error {
	return nil
}
