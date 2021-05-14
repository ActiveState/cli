package request

type AvailableUpdate struct {
}

func NewAvailableUpdate() *AvailableUpdate {
	return &AvailableUpdate{}
}

func (u *AvailableUpdate) Query() string {
	return `query() {
		availableUpdate() {
			available
		}
	}`
}

func (u *AvailableUpdate) Vars() map[string]interface{} {
	return nil
}
