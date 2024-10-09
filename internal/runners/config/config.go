package config

import (
	"fmt"
	"sort"

	"github.com/ActiveState/cli/internal/config"
	"github.com/ActiveState/cli/internal/locale"
	mediator "github.com/ActiveState/cli/internal/mediators/config"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/primer"
	"github.com/ActiveState/cli/internal/table"
)

type List struct {
	out output.Outputer
	cfg *config.Instance
}

type primeable interface {
	primer.Outputer
	primer.Configurer
	primer.SvcModeler
	primer.Analyticer
}

func NewList(prime primeable) (*List, error) {
	return &List{
		out: prime.Output(),
		cfg: prime.Config(),
	}, nil
}

type structuredConfigData struct {
	Key     string      `json:"key"`
	Value   interface{} `json:"value"`
	Default interface{} `json:"default"`
	opt     mediator.Option
}

func (c *List) Run(usageFunc func() error) error {
	registered := mediator.AllRegistered()
	sort.SliceStable(registered, func(i, j int) bool {
		return registered[i].Name < registered[j].Name
	})

	var data []structuredConfigData
	for _, opt := range registered {
		configuredValue := c.cfg.Get(opt.Name)
		data = append(data, structuredConfigData{
			Key:     opt.Name,
			Value:   configuredValue,
			Default: mediator.GetDefault(opt),
			opt:     opt,
		})
	}

	if c.out.Type().IsStructured() {
		c.out.Print(output.Structured(data))
	} else {
		if err := c.renderUserFacing(data); err != nil {
			return err
		}
	}

	return nil
}

func (c *List) renderUserFacing(configData []structuredConfigData) error {
	tbl := table.New(locale.Ts("Key", "Value", "Default"))
	tbl.HideDash = true
	for _, config := range configData {
		tbl.AddRow([]string{
			formatKey(config.Key),
			formatValue(config.opt, config.Value),
			formatDefault(config.Default),
		})
	}

	c.out.Print(tbl.Render())
	c.out.Print("")
	c.out.Print(locale.T("config_get_help"))
	c.out.Print(locale.T("config_set_help"))

	return nil
}

func formatKey(key string) string {
	return fmt.Sprintf("[CYAN]%s[/RESET]", key)
}

func formatValue(opt mediator.Option, value interface{}) string {
	var v string
	switch opt.Type {
	case mediator.Enum:
		return fmt.Sprintf("\"%s\"", value)
	default:
		v = fmt.Sprintf("%v", value)
	}

	if v == "" {
		return "\"\""
	}

	if len(v) > 100 {
		v = v[:100] + "..."
	}

	if value == mediator.GetDefault(opt) {
		return fmt.Sprintf("[GREEN]%s[/RESET]", v)
	}

	return fmt.Sprintf("[BOLD][RED]%s*[/RESET]", v)
}

func formatDefault(defaultValue interface{}) string {
	v := fmt.Sprintf("%v", defaultValue)
	if v == "" {
		v = "\"\""
	}
	return fmt.Sprintf("[DISABLED]%s[/RESET]", v)
}
