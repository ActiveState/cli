package tracker

type EnvironmentVariable struct {
	Key   string
	Value string
	label string
}

func (ev *EnvironmentVariable) Type() TrackingType {
	return Environment
}

func (ev *EnvironmentVariable) Label() string {
	return ev.label
}
