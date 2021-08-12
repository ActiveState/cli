package model

// SupportedLanguagesResponse is a struct for the payload of the supported languages mediator endpoint
type SupportedLanguagesResponse struct {
	Languages []SupportedLanguage `json:"unstableSupportedLanguages"`
}

type SupportedLanguage struct {
	Name           string `json:"name"`
	DefaultVersion string `json:"default_version"`
}
