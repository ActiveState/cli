package request

type AvailableUpdate struct {
	channel string
	version string
}

func NewAvailableUpdate(channel, version string) *AvailableUpdate {
	return &AvailableUpdate{
		channel: channel,
		version: version,
	}
}

func (u *AvailableUpdate) Query() string {
	return `query($channel: String!, $version: String!) {
	availableUpdate(channel: $channel, version: $version) {
			channel
			version
			path
			platform
			sha256
		}
	}`
}

func (u *AvailableUpdate) Vars() map[string]interface{} {
	return map[string]interface{}{
		"channel": u.channel,
		"version": u.version,
	}
}
