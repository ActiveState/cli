package project

import (
	"os"
	"strings"

	"github.com/ActiveState/cli/internal/constants"
)

type EventType string

const (
	BeforeCmd     EventType = "before-command"
	AfterCmd      EventType = "after-command"
	Activate      EventType = "activate"
	FirstActivate EventType = "first-activate"
)

func (e EventType) String() string {
	return string(e)
}

func ActivateEvents() []EventType {
	if strings.EqualFold(os.Getenv(constants.DisableActivateEventsEnvVarName), "true") {
		return []EventType{}
	}

	return []EventType{
		Activate,
		FirstActivate,
	}
}
