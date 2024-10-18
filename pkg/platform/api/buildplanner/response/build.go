package response

import (
	"encoding/json"

	"github.com/go-openapi/strfmt"
)

type ArtifactResponse struct {
	NodeID      strfmt.UUID `json:"nodeId"`
	Errors      []string    `json:"errors"`
	Status      string      `json:"status"`
	DisplayName string      `json:"displayName"`
	LogURL      string      `json:"logURL"`
}

type BuildResponse struct {
	*Error
	*PlanningError
	Type       string             `json:"__typename"`
	Artifacts  []ArtifactResponse `json:"artifacts"`
	Status     string             `json:"status"`
	RawMessage json.RawMessage    `json:"rawMessage"`
}

func (b *BuildResponse) MarshalJSON() ([]byte, error) {
	return b.RawMessage.MarshalJSON()
}

// UnmarshalJSON lets us record both the raw json message as well as unmarshal the parts we care about
// because without this function only the RawMessage itself would be set, the rest of the field would be empty.
// This is effectively working around a silly json library limitation.
func (b *BuildResponse) UnmarshalJSON(data []byte) error {
	type Alias BuildResponse
	aux := &struct {
		*Alias
	}{
		Alias: (*Alias)(b),
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	b.RawMessage = data
	return nil
}
