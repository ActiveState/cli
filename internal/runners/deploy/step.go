package deploy

import (
	"strings"

	"github.com/ActiveState/cli/internal/captain"
	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/internal/locale"
)

// Step is the --step flag for the --deploy command, it implements captain.FlagMarshaler
type Step int

var _ captain.FlagMarshaler = func(t Step) *Step { return &t }(0)

const (
	UnsetStep Step = iota
	InstallStep
	ConfigureStep
	ReportStep
)

var StepMap = map[Step]string{
	UnsetStep:     "unset",
	InstallStep:   "install",
	ConfigureStep: "configure",
	ReportStep:    "report",
}

func (t Step) String() string {
	for k, v := range StepMap {
		if k == t {
			return v
		}
	}
	return StepMap[UnsetStep]
}

func (t *Step) Set(value string) error {
	for k, v := range StepMap {
		if v == value && k != UnsetStep {
			*t = k
			return nil
		}
	}

	return failures.FailInput.New(locale.Tr("err_invalid_step",
		value,
		strings.Join([]string{InstallStep.String(), ConfigureStep.String(), ReportStep.String()}, ", "), // allowed values
	))
}

func (t *Step) Type() string {
	return "step"
}
