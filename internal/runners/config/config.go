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

type structuredConfigData struct {
	Key     string      `json:"key"`
	Value   interface{} `json:"value"`
	Default interface{} `json:"default"`
	opt     mediator.Option
}

func (c *List) Run() error {
	registered := mediator.Registered()

	cfg := c.prime.Config()
	out := c.prime.Output()

	var data []structuredConfigData
	for _, opt := range registered {
		configuredValue := cfg.Get(opt.Name)
		data = append(data, structuredConfigData{
			Key:     opt.Name,
			Value:   configuredValue,
			Default: mediator.GetDefault(opt),
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

	tbl := table.New(locale.Ts("key", "value", "default"))
	tbl.HideDash = true
	for _, config := range configData {
		tbl.AddRow([]string{
			fmt.Sprintf("[CYAN]%s[/RESET]", config.Key),
			renderConfigValue(cfg, config.opt),
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
	configured := cfg.Get(opt.Name)
	var tags []string
	if opt.Type == mediator.Bool {
		if configured == true {
			tags = append(tags, "[GREEN]")
		} else {
			tags = append(tags, "[RED]")
		}
	}

	value := formatValue(opt, configured)
	if cfg.IsSet(opt.Name) {
		tags = append(tags, "[BOLD]")
		value = value + "*"
	}

	if len(tags) > 0 {
		return fmt.Sprintf("%s%s[/RESET]", strings.Join(tags, ""), value)
	}

	return value
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
