package config

import (
	"fmt"
	"sort"

	"github.com/ActiveState/cli/internal/config"
	"github.com/ActiveState/cli/internal/locale"
	mediator "github.com/ActiveState/cli/internal/mediators/config"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/primer"
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

type configData struct {
	Key     string `locale:"key,Key"`
	Value   string `locale:"value,Value"`
	Default string `locale:"default,Default"`
}

type configOutput struct {
	out     output.Outputer
	cfg     *config.Instance
	options []mediator.Option
	data    []configData
}

func (c *List) Run(usageFunc func() error) error {
	registered := mediator.AllRegistered()
	sort.SliceStable(registered, func(i, j int) bool {
		return registered[i].Name < registered[j].Name
	})

	var data []configData
	for _, opt := range registered {
		configuredValue := c.cfg.Get(opt.Name)
		data = append(data, configData{
			Key:     formatKey(opt.Name),
			Value:   formatValue(opt, configuredValue),
			Default: formatDefault(mediator.GetDefault(opt)),
		})
	}

	c.out.Print(&configOutput{
		out:     c.out,
		cfg:     c.cfg,
		options: registered,
		data:    data,
	})

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

	if value == opt.Default {
		return fmt.Sprintf("[GREEN]%s[/RESET]", v)
	}

	return fmt.Sprintf("[BOLD][RED]%s*[/RESET]", v)
}

func formatDefault[T any](defaultValue T) string {
	v := fmt.Sprintf("%v", defaultValue)
	if v == "" {
		v = "\"\""
	}
	return fmt.Sprintf("[DISABLED]%s[/RESET]", v)
}

func (c *configOutput) MarshalOutput(format output.Format) interface{} {
	if format != output.PlainFormatName {
		return c.data
	}

	c.out.Print(struct {
		Data []configData `opts:"table,hideDash,omitKey"`
	}{c.data})
	c.out.Print("")
	c.out.Print(locale.T("config_get_help"))
	c.out.Print(locale.T("config_set_help"))

	return output.Suppress
}

func (c *configOutput) MarshalStructured(format output.Format) interface{} {
	return c.data
}
