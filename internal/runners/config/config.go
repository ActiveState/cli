package config

import (
	"fmt"
	"strings"

	"github.com/ActiveState/cli/internal/config"
	"github.com/ActiveState/cli/internal/locale"
	mediator "github.com/ActiveState/cli/internal/mediators/config"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/primer"
	"github.com/ActiveState/cli/internal/table"
)

type List struct {
	prime primeable
}

type primeable interface {
	primer.Outputer
	primer.Configurer
	primer.SvcModeler
	primer.Analyticer
}

func NewList(prime primeable) (*List, error) {
	return &List{
		prime: prime,
	}, nil
}

// Where an option's effective value comes from.
const (
	sourceEnvironment = "environment" // overridden by an environment variable
	sourceLocal       = "local"       // set locally by the user (stored in config.db)
	sourceDefault     = "default"     // neither set nor overridden; using the built-in default
)

type structuredConfigData struct {
	Key     string      `json:"key"`
	Value   interface{} `json:"value"`
	Default interface{} `json:"default"`
	// Source is where the effective value comes from: "environment", "local", or "default".
	Source string `json:"source"`
	// EnvVar is the canonical environment variable that can override this key (always present).
	EnvVar string `json:"envVar"`
	// Env is the name of the environment variable currently overriding this value, if any.
	Env string `json:"env,omitempty"`
	opt mediator.Option
}

// effectiveValue returns the value the State Tool will actually use for the option, along with the
// name of the environment variable overriding it (empty when the value comes from stored config or
// the built-in default).
func effectiveValue(cfg *config.Instance, opt mediator.Option) (interface{}, string) {
	if envValue, envVar, ok := mediator.EnvOverride(opt); ok {
		return envValue, envVar
	}
	return cfg.Get(opt.Name), ""
}

// configSource reports where an option's effective value comes from, and the name of the
// environment variable in effect when the source is the environment.
func configSource(cfg *config.Instance, opt mediator.Option) (source string, envVar string) {
	if _, name, ok := mediator.EnvOverride(opt); ok {
		return sourceEnvironment, name
	}
	if cfg.IsSet(opt.Name) {
		return sourceLocal, ""
	}
	return sourceDefault, ""
}

func (c *List) Run() error {
	registered := mediator.Registered()

	cfg := c.prime.Config()
	out := c.prime.Output()

	var data []structuredConfigData
	for _, opt := range registered {
		configuredValue, envVar := effectiveValue(cfg, opt)
		source, _ := configSource(cfg, opt)
		data = append(data, structuredConfigData{
			Key:     opt.Name,
			Value:   configuredValue,
			Default: mediator.GetDefault(opt),
			Source:  source,
			EnvVar:  mediator.CanonicalEnvVarName(opt.Name),
			Env:     envVar,
			opt:     opt,
		})
	}

	if out.Type().IsStructured() {
		out.Print(output.Structured(data))
	} else {
		if err := c.renderUserFacing(data); err != nil {
			return err
		}
	}

	return nil
}

func (c *List) renderUserFacing(configData []structuredConfigData) error {
	cfg := c.prime.Config()
	out := c.prime.Output()

	tbl := table.New(locale.Ts("key", "value", "source", "default"))
	tbl.HideDash = true
	for _, config := range configData {
		tbl.AddRow([]string{
			fmt.Sprintf("[CYAN]%s[/RESET]", config.Key),
			renderConfigValue(cfg, config.opt),
			renderConfigSource(cfg, config.opt),
			fmt.Sprintf("[DISABLED]%s[/RESET]", formatValue(config.opt, config.Default)),
		})
	}

	out.Print(tbl.Render())
	out.Notice("")
	out.Notice(locale.T("config_get_help"))
	out.Notice(locale.T("config_set_help"))

	return nil
}

func renderConfigValue(cfg *config.Instance, opt mediator.Option) string {
	configured, _ := effectiveValue(cfg, opt)
	var tags []string
	if opt.Type == mediator.Bool {
		if configured == true {
			tags = append(tags, "[GREEN]")
		} else {
			tags = append(tags, "[RED]")
		}
	}

	value := formatValue(opt, configured)
	if len(tags) > 0 {
		return fmt.Sprintf("%s%s[/RESET]", strings.Join(tags, ""), value)
	}

	return value
}

// renderConfigSource renders the Source column: the environment variable in effect (when overridden
// by the environment), "local" (set by the user), or a de-emphasized "default".
func renderConfigSource(cfg *config.Instance, opt mediator.Option) string {
	source, envVar := configSource(cfg, opt)
	switch source {
	case sourceEnvironment:
		return fmt.Sprintf("[BOLD]%s[/RESET]", envVar)
	case sourceLocal:
		return fmt.Sprintf("[BOLD]%s[/RESET]", locale.Tl("config_source_local", "local"))
	default:
		return fmt.Sprintf("[DISABLED]%s[/RESET]", locale.Tl("config_source_default", "default"))
	}
}

func formatValue(opt mediator.Option, value interface{}) string {
	switch opt.Type {
	case mediator.Enum, mediator.String:
		return formatString(fmt.Sprintf("%v", value))
	default:
		return fmt.Sprintf("%v", value)
	}
}

func formatString(value string) string {
	if value == "" {
		return "\"\""
	}

	if len(value) > 100 {
		value = value[:100] + "..."
	}

	return fmt.Sprintf("\"%s\"", value)
}
