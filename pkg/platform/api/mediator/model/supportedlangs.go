package model

// SupportedLanguagesResponse is a struct for the payload of the supported languages mediator endpoint
type SupportedLanguagesResponse struct {
	Languages []string `json:"unstableSupportedLanguages"`
}
