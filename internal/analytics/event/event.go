package event

import "github.com/ActiveState/cli/internal/output"

// EventData is the data that is synchronized on disk (or forwarded to the state-svc server)
type EventData struct {
	Category    string `json:"category"`
	Action      string `json:"action"`
	Label       string `json:"label"`
	ProjectName string `json:"project"`
	Output      string `json:"output_type"`
	UserID      string `json:"user_id"`
}

func New(category, action string, label, projectName, out, userID *string) EventData {
	lbl := ""
	if label != nil {
		lbl = *label
	}

	pn := ""
	if projectName != nil {
		pn = *projectName
	}

	o := string(output.PlainFormatName)
	if out != nil {
		o = *out
	}

	uid := ""
	if userID != nil {
		uid = *userID
	}

	return EventData{
		category,
		action,
		lbl,
		pn,
		o,
		uid,
	}
}
