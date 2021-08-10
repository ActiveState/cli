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
	return `query ($os_name: String!) {
		unstableSupportedLanguages(os_name: $os_name) {
			name
			default_version
		}
	}`
}

func (p *supportedLanguages) Vars() map[string]interface{} {
	return p.vars
}
