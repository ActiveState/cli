package model

// SupportedLanguage is a struct for the payload of the supported languages mediator endpoint
type SupportedLanguage struct {
	Name           string `json:"name"`
	DefaultVersion string `json:"default_version"`
}
