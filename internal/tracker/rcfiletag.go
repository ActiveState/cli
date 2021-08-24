package tracker

type RCFileTag struct {
	key   string
	value string
}

func NewRCFileTag(key, value string) RCFileTag {
	return RCFileTag{key, value}
}

func (r RCFileTag) Type() TrackingType {
	return FileTag
}

func (r RCFileTag) Key() string {
	return r.key
}

func (r RCFileTag) Value() string {
	return r.value
}
