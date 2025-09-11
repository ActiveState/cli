package response

import (
	"github.com/go-openapi/strfmt"
)

type EvaluateResponse struct {
	Evaluate `json:"evaluate"`
}

type Evaluate struct {
	Type      string      `json:"__typename"`
	Status    string      `json:"status"`
	SessionID strfmt.UUID `json:"sessionId"`
	*Error
	*ErrorWithSubErrors
}
