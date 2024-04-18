package response

import (
	"encoding/json"
)

type BuildResponse struct {
	json.RawMessage
	Artifacts []struct {
		Errors []string `json:"errors"`
	} `json:"artifacts"`
	Status string `json:"status"`
	*Error
	*PlanningError
}
