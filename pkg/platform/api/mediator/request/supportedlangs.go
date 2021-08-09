package request

func SupportedLanguages(osName string) *supportedLanguages {
	return &supportedLanguages{map[string]interface{}{
		"os_name": osName,
	}}
}

type supportedLanguages struct {
	vars map[string]interface{}
}

func (p *supportedLanguages) Query() string {
	return `query {
		unstableSupportedLanguages($os_name: String!)
		{
			name
			default_version
		}
	}`
}

func (p *supportedLanguages) Vars() map[string]interface{} {
	return p.vars
}
