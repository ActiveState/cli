package track

import "gopkg.in/yaml.v3"

type RCEntry struct {
	Start string `yaml:"start"`
	End   string `yaml:"end"`
}

func NewRCEntry(start, end string) *RCEntry {
	return &RCEntry{start, end}
}

func (r *RCEntry) Type() TrackingType {
	return RCEntryType
}

type RCEntries []*RCEntry

func (r RCEntries) UnmarshalTrackable(value string) error {
	return yaml.Unmarshal([]byte(value), &r)
}
