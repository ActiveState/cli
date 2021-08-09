package request

func SupportedLanguages() *supportedLanguages {
	return &supportedLanguages{}
}

type supportedLanguages struct {
}

func (p *supportedLanguages) Query() string {
	return `query {
		unstableSupportedLanguages()
		{
			name
			default_version
		}
	}`
}

func (p *supportedLanguages) Vars() map[string]interface{} {
	return nil
}
