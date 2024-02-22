package deploy

import (
	"strings"

	"github.com/ActiveState/cli/internal/locale"
)

// Step is the --step flag for the --deploy command, it implements captain.FlagMarshaler
type Step int

const (
	UnsetStep Step = iota
	InstallStep
	ConfigureStep
	SymlinkStep
	ReportStep
)

var StepMap = map[Step]string{
	UnsetStep:     "unset",
	InstallStep:   "install",
	ConfigureStep: "configure",
	SymlinkStep:   "symlink",
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

	return locale.NewInputError("err_invalid_step", "",
		value,
		strings.Join([]string{InstallStep.String(), ConfigureStep.String(), ReportStep.String()}, ", "), // allowed values
	)
}

func (t *Step) Type() string {
	return "value"
}
