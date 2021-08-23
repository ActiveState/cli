package request

type AvailableUpdate struct {
}

func NewAvailableUpdate() *AvailableUpdate {
	return &AvailableUpdate{}
}

func (u *AvailableUpdate) Query() string {
	return `query() {
		availableUpdate() {
			version
			channel
			path
			platform
			sha256
			tag
			url
		}
	}`
}

func (u *AvailableUpdate) Vars() map[string]interface{} {
	return nil
}
