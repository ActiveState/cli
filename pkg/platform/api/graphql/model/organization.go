package model

import "github.com/go-openapi/strfmt"

type Organizations struct {
	Organizations []Organization `json:"organizations"`
}

type Organization struct {
	ID          strfmt.UUID `json:"organization_id"`
	DisplayName string      `json:"display_name"`
	URLName     string      `json:"url_name"`
}
