package request

type AvailableUpdate struct {
	desiredChannel string
	desiredVersion string
}

func NewAvailableUpdate(desiredChannel, desiredVersion string) *AvailableUpdate {
	return &AvailableUpdate{
		desiredChannel: desiredChannel,
		desiredVersion: desiredVersion,
	}
}

func (u *AvailableUpdate) Query() string {
	return `query($desiredChannel: String!, $desiredVersion: String!) {
	availableUpdate(desiredChannel: $desiredChannel, desiredVersion: $desiredVersion) {
			channel
			version
			path
			platform
			sha256
		}
	}`
}

func (u *AvailableUpdate) Vars() (map[string]interface{}, error) {
	return map[string]interface{}{
		"desiredChannel": u.desiredChannel,
		"desiredVersion": u.desiredVersion,
	}, nil
}
