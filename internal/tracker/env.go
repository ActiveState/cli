package tracker

type EnvironmentVariable struct {
	key   string
	value string
}

func NewEnvironmentVariable(key, value string) EnvironmentVariable {
	return EnvironmentVariable{key, value}
}

func (ev EnvironmentVariable) Type() TrackingType {
	return Environment
}

func (ev EnvironmentVariable) Key() string {
	return ev.key
}

func (ev EnvironmentVariable) Value() string {
	return ev.value
}
